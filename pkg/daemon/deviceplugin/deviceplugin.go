package deviceplugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"gopkg.in/fsnotify.v1"
	devicepluginv1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	devicePluginsDir   = "/var/lib/kubelet/device-plugins"
	kubeletSocketName  = "kubelet.sock"
	socketNamePrefix   = "virtink-"
	resourceNamePrefix = "devices.virtink.io/"
)

type devicePluginManager struct {
	devicePlugins []*devicePlugin
}

func NewDevicePluginManager() *devicePluginManager {
	return &devicePluginManager{
		devicePlugins: []*devicePlugin{
			newDevicePlugin("kvm", "/dev/kvm", 1000),
			newDevicePlugin("tun", "/dev/net/tun", 1000),
		},
	}
}

func (dpm *devicePluginManager) Start(ctx context.Context) error {
	for _, dp := range dpm.devicePlugins {
		dp.Start()
	}

	<-ctx.Done()

	for _, dp := range dpm.devicePlugins {
		dp.Stop()
	}
	return nil
}

type devicePlugin struct {
	deviceName string
	devicePath string
	devices    []*devicepluginv1beta1.Device
	socketPath string
	server     *grpc.Server
	health     chan string
}

func newDevicePlugin(deviceName string, devicePath string, deviceCount int) *devicePlugin {
	dp := &devicePlugin{
		deviceName: deviceName,
		devicePath: devicePath,
	}
	for i := 1; i <= deviceCount; i++ {
		dp.devices = append(dp.devices, &devicepluginv1beta1.Device{
			ID:     deviceName + strconv.Itoa(i),
			Health: devicepluginv1beta1.Healthy,
		})
	}
	return dp
}

func (dp *devicePlugin) Start() {
	go func() {
		for {
			if err := dp.run(); err != nil {
				ctrl.Log.Error(err, "run device plugin", "device", dp.deviceName)
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func (dp *devicePlugin) run() error {
	dp.server = grpc.NewServer()
	devicepluginv1beta1.RegisterDevicePluginServer(dp.server, dp)

	dp.socketPath = filepath.Join(devicePluginsDir, fmt.Sprintf("%s%s.sock", socketNamePrefix, dp.deviceName))
	_ = os.Remove(dp.socketPath)
	listener, err := net.Listen("unix", dp.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen %s: %s", dp.socketPath, err)
	}
	defer listener.Close()

	errChan := make(chan error, 1)
	go func() {
		errChan <- dp.server.Serve(listener)
	}()
	if err := waitForGRPCServer(dp.socketPath, 5*time.Second); err != nil {
		return err
	}

	if err := dp.register(); err != nil {
		return err
	}

	go func() {
		errChan <- dp.healthCheck()
	}()

	err = <-errChan
	return err
}

func (dp *devicePlugin) register() error {
	kubeletSocket := filepath.Join(devicePluginsDir, kubeletSocketName)
	conn, err := grpc.Dial(fmt.Sprintf("unix://%s", kubeletSocket), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("dial %s: %s", kubeletSocket, err)
	}
	defer conn.Close()

	client := devicepluginv1beta1.NewRegistrationClient(conn)
	req := &devicepluginv1beta1.RegisterRequest{
		Version:      "v1beta1",
		Endpoint:     filepath.Base(dp.socketPath),
		ResourceName: fmt.Sprintf("%s%s", resourceNamePrefix, dp.deviceName),
	}
	if _, err := client.Register(context.Background(), req); err != nil {
		return fmt.Errorf("register to kubelet: %s", err)
	}
	return nil
}

func (dp *devicePlugin) Stop() {
	dp.server.Stop()
	os.RemoveAll(dp.socketPath)
}

func (dp *devicePlugin) GetDevicePluginOptions(ctx context.Context, req *devicepluginv1beta1.Empty) (*devicepluginv1beta1.DevicePluginOptions, error) {
	return &devicepluginv1beta1.DevicePluginOptions{}, nil
}

func (dp *devicePlugin) ListAndWatch(req *devicepluginv1beta1.Empty, stream devicepluginv1beta1.DevicePlugin_ListAndWatchServer) error {
	resp := &devicepluginv1beta1.ListAndWatchResponse{Devices: dp.devices}
	stream.Send(resp)

	tick := time.NewTicker(30 * time.Second)
	for {
		select {
		case health := <-dp.health:
			for _, dev := range dp.devices {
				dev.Health = health
			}
			resp := &devicepluginv1beta1.ListAndWatchResponse{Devices: dp.devices}
			if err := stream.Send(resp); err != nil {
				return fmt.Errorf("send response: %s", err)
			}
		case <-tick.C:
			resp := &devicepluginv1beta1.ListAndWatchResponse{Devices: dp.devices}
			if err := stream.Send(resp); err != nil {
				return fmt.Errorf("send response: %s", err)
			}
		}
	}
}

func (dp *devicePlugin) Allocate(ctx context.Context, req *devicepluginv1beta1.AllocateRequest) (*devicepluginv1beta1.AllocateResponse, error) {
	var containerResps []*devicepluginv1beta1.ContainerAllocateResponse
	for _, containerReq := range req.ContainerRequests {
		var devices []*devicepluginv1beta1.DeviceSpec
		for i := 0; i < len(containerReq.DevicesIDs); i++ {
			devices = append(devices, &devicepluginv1beta1.DeviceSpec{
				HostPath:      dp.devicePath,
				ContainerPath: dp.devicePath,
				Permissions:   "rwm",
			})
		}

		containerResp := &devicepluginv1beta1.ContainerAllocateResponse{
			Devices: devices,
		}
		containerResps = append(containerResps, containerResp)
	}
	return &devicepluginv1beta1.AllocateResponse{
		ContainerResponses: containerResps,
	}, nil
}

func (dp *devicePlugin) GetPreferredAllocation(context.Context, *devicepluginv1beta1.PreferredAllocationRequest) (*devicepluginv1beta1.PreferredAllocationResponse, error) {
	return nil, nil
}

func (dp *devicePlugin) PreStartContainer(context.Context, *devicepluginv1beta1.PreStartContainerRequest) (*devicepluginv1beta1.PreStartContainerResponse, error) {
	return nil, nil
}

func waitForGRPCServer(socketPath string, timeout time.Duration) error {
	c, err := grpc.Dial(socketPath,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
	if err != nil {
		return err
	}
	c.Close()
	return nil
}

func (dp *devicePlugin) healthCheck() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %s", err)
	}
	defer watcher.Close()

	if err := watcher.Add(filepath.Dir(dp.devicePath)); err != nil {
		return fmt.Errorf("start watching %s: %s", dp.devicePath, err)
	}

	if err := watcher.Add(filepath.Dir(dp.socketPath)); err != nil {
		return fmt.Errorf("start watching %s: %s", dp.socketPath, err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				continue
			}
			if dp.devicePath == event.Name {
				if event.Op == fsnotify.Remove || event.Op == fsnotify.Rename {
					ctrl.Log.Info("device file was removed", "device", dp.devicePath)
					dp.health <- devicepluginv1beta1.Unhealthy
				} else if event.Op == fsnotify.Create {
					ctrl.Log.Info("device file was created", "device", dp.devicePath)
					dp.health <- devicepluginv1beta1.Healthy
				}
			} else if dp.socketPath == event.Name && event.Op == fsnotify.Remove {
				ctrl.Log.Info("device socket file was removed", "socket", dp.socketPath)
				return nil
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				continue
			}
			ctrl.Log.Error(err, "watching device file and socket", "device", dp.deviceName)
		}
	}
}
