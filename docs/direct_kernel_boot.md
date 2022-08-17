# Direct Kernel Boot

Virtink uses Cloud Hypervisor as the underlying hypervisor for VMs. Cloud Hypervisor supports direct kernel boot into a `vmlinux` ELF kernel. It should work with a rootfs from most distributions and does not require an EFI system partition. To use the direct kernel boot feature in Virtink, both a kernel image and a rootfs volume are required.

## Kernel Images

In order to support virtio-watchdog, Cloud Hypervisor has its own kernel development branch. Currently, we provide a pre-built v5.15.12 kernel image in the Docker Hub. In the future, more versions of pre-built kernels may be added.

Kernel is configured in `spec.instance.kernel`. You should specify both the `image` name of the kernel, and the Linux `cmdline` to start the kernel. Below is an example to use the pre-built kernel image for direct kernel booting:

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    kernel:
      image: smartxworks/virtink-kernel-5.15.12
      cmdline: "console=ttyS0 root=/dev/vda rw"
    disks:
      - name: ubuntu
  volumes:
    - name: ubuntu
      containerRootfs:
        image: smartxworks/virtink-container-rootfs-ubuntu
        size: 4Gi
```

### Building and Using Your Own Kernel

It's possible to build and use your own kernel for direct kernel booting. To build a kernel that can work properly with Cloud Hypervisor, refer to the [Cloud Hypervisor documentation](https://github.com/cloud-hypervisor/cloud-hypervisor#building-your-kernel).

After successfully built your kernel, you should have a `vmlinux` file ready to be injected into a container image. The `vmlinux` file must be placed at exactly the `/vmlinux` path and the image must be based on `smartxworks/virtink-kernel-base`.

Below is an example of injecting a local kernel into a container image:

```dockerfile
FROM smartxworks/virtink-kernel-base
COPY vmlinux /vmlinux
```

## Rootfs Volumes

The rootfs defines the root filesystem of the VM. The root parition from most distributions should work for direct kernel booting. However, Virtink does provide a more effortless way to build and use a rootfs using Docker with the `containerRootfs` volume feature.

### `containerRootfs` Volume

The `containerRootfs` feature provides the ability to store and distribute VM rootfs in the container image registry. Everything you need to build a `containerRootfs` image is the Docker toolchain. No raw or QCOW2 images are involved. For building and using a `containerRootfs` image with direct kernel booting, refer to the [`containerRootfs` volume documentation](disks_and_volumes.md#containerrootfs-volume).

### Other Types of Volumes

Other types of [volumes](disks_and_volumes.md#volumes) can also be a valid rootfs source for direct kernel booting, as long as the containing disk image has a root partition and you specify it correctly in the `spec.instance.kernel.cmdline`. Below is an example of direct kernel booting a Ubuntu VM using the Ubuntu cloud image inside a `containerDisk` volume:

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    kernel:
      image: smartxworks/virtink-kernel-5.15.12
      cmdline: "console=ttyS0 root=/dev/vda1 rw"
    disks:
      - name: ubuntu
  volumes:
    - name: ubuntu
      containerDisk:
        image: smartxworks/virtink-container-disk-ubuntu
```
