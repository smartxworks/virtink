package sanlock

import (
	"fmt"
	"strings"
	"unsafe"
)

/*
#cgo LDFLAGS: -lsanlock -lsanlock_client
#include <stdint.h>
#include <stdlib.h>
#include <libaio.h>
#include <errno.h>
#include "sanlock.h"
#include "sanlock_direct.h"
#include "sanlock_admin.h"
#include "sanlock_resource.h"

struct aicb {
	int used;
	char *buf;
	struct iocb iocb;
};

#define NAME_ID_SIZE 48

struct task {
	char name[NAME_ID_SIZE+1];   // for log messages

	unsigned int io_count;       // stats
	unsigned int to_count;       // stats

	int use_aio;
	int cb_size;
	char *iobuf;
	io_context_t aio_ctx;
	struct aicb *read_iobuf_timeout_aicb;
	struct aicb *callbacks;
};

int direct_rindex_format(struct task *task, struct sanlk_rindex *ri);
*/
import "C"

const (
	OffsetLockspace = 0
	OffsetRIndex    = 1048576

	WatchdogFireTimeoutDefaultSeconds = 60
	IOTimeoutDefaultSeconds           = 10
	IDRenewalDefaultSeconds           = 2 * IOTimeoutDefaultSeconds
	IDRenewalFailDefaultSeconds       = 4 * IDRenewalDefaultSeconds
	HostDeadDefaultSeconds            = IDRenewalFailDefaultSeconds + WatchdogFireTimeoutDefaultSeconds
)

type ErrorNumber int

const (
	// TODO: more error
	EPERM       = ErrorNumber(C.EPERM)
	ENOENT      = ErrorNumber(C.ENOENT)
	EINVAL      = ErrorNumber(C.EINVAL)
	EEXIST      = ErrorNumber(C.EEXIST)
	EINPROGRESS = ErrorNumber(C.EINPROGRESS)
	EAGAIN      = ErrorNumber(C.EAGAIN)
)

func (e ErrorNumber) Error() string {
	var err string
	switch e {
	case EPERM:
		err = "EPERM"
	case ENOENT:
		err = "ENOENT"
	case EINVAL:
		err = "EINVAL"
	case EEXIST:
		err = "EEXIST"
	case EINPROGRESS:
		err = "EINPROGRESS"
	case EAGAIN:
		err = "EAGAIN"
	}
	return fmt.Sprintf("errCode(%d): %s", e, err)
}

type HostStatusFlag uint32

const (
	HostStatusUnknown = HostStatusFlag(C.SANLK_HOST_UNKNOWN)
	HostStatusFree    = HostStatusFlag(C.SANLK_HOST_FREE)
	HostStatusLive    = HostStatusFlag(C.SANLK_HOST_LIVE)
	HostStatusFail    = HostStatusFlag(C.SANLK_HOST_FAIL)
	HostStatusDead    = HostStatusFlag(C.SANLK_HOST_DEAD)
)

func WriteLockspace(lockspace string, path string) error {
	return WriteLockspaceWithIOTimeout(lockspace, path, 0)
}

func WriteLockspaceWithIOTimeout(lockspace string, path string, ioTimeout uint32) error {
	ls := buildSanlockLockspace(lockspace, path, 0)

	if rv := C.sanlock_direct_write_lockspace(&ls, C.int(2000), 0, C.uint(ioTimeout)); rv < 0 {
		return ErrorNumber(-rv)
	}
	return nil
}

func FormatRIndex(lockspace string, path string) error {
	rIndex := buildSanlockRIndex(lockspace, path)

	if rv := C.direct_rindex_format(&C.struct_task{}, &rIndex); rv < 0 {
		return ErrorNumber(-rv)
	}
	return nil
}

func CreateResource(lockspace string, path string, resource string) (uint64, error) {
	rIndex := buildSanlockRIndex(lockspace, path)
	rEntry := buildSanlockREntry(resource)

	if rv := C.sanlock_create_resource(&rIndex, 0, &rEntry, 0, 0); rv < 0 {
		return 0, ErrorNumber(-rv)
	}
	return uint64(rEntry.offset), nil
}

func DeleteResource(lockspace string, path string, resource string) error {
	rIndex := buildSanlockRIndex(lockspace, path)
	rEntry := buildSanlockREntry(resource)

	if rv := C.sanlock_delete_resource(&rIndex, 0, &rEntry); rv < 0 {
		return ErrorNumber(-rv)
	}
	return nil
}

func SearchResource(lockspace string, path string, resource string) (uint64, error) {
	rIndex := buildSanlockRIndex(lockspace, path)
	rEntry := buildSanlockREntry(resource)

	if rv := C.sanlock_lookup_rindex(&rIndex, 0, &rEntry); rv < 0 {
		return 0, ErrorNumber(-rv)
	}
	return uint64(rEntry.offset), nil
}

// AcquireDeltaLease returns:
// nil: acquire delta lease successfully
// EEXIST: the lockspace already exists
// EINPROGRESS: the lockspace is already in the process of being added (the in-progress add may or may not succeed)
// EAGAIN: the lockspace is being removed
func AcquireDeltaLease(lockspace string, path string, id uint64) error {
	if id < 1 || id > 2000 {
		return fmt.Errorf("invalid host ID, allowed value 1~2000")
	}

	ls := buildSanlockLockspace(lockspace, path, id)

	if rv := C.sanlock_add_lockspace(&ls, 0); rv < 0 {
		return ErrorNumber(-rv)
	}
	return nil
}

// ReleaseDeltaLease returns:
// EINPROGRESS: the lockspace is already in the process of being removed
// ENOENT: lockspace not found
//
// The sanlock daemon will kill any pids using the lockspace when the
// lockspace is removed.
func ReleaseDeltaLease(lockspace string, path string, id uint64) error {
	if id < 1 || id > 2000 {
		return fmt.Errorf("invalid host ID, allowed value 1~2000")
	}

	ls := buildSanlockLockspace(lockspace, path, id)

	if rv := C.sanlock_rem_lockspace(&ls, 0); rv < 0 {
		return ErrorNumber(-rv)
	}
	return nil
}

func HasDeltaLease(lockspace string, path string, id uint64) bool {
	if id < 1 || id > 2000 {
		return false
	}

	ls := buildSanlockLockspace(lockspace, path, id)

	return C.sanlock_inq_lockspace(&ls, 0) == 0
}

func GetHostStatus(lockspace string, id uint64) (HostStatusFlag, error) {
	if id < 1 || id > 2000 {
		return 0, fmt.Errorf("invalid host ID, allowed value 1~2000")
	}

	var host *C.struct_sanlk_host
	var num C.int
	lockspaceName := C.CString(lockspace)
	defer C.free(unsafe.Pointer(lockspaceName))
	if rv := C.sanlock_get_hosts(lockspaceName, C.ulong(id), &host, &num, 0); rv < 0 {
		return 0, ErrorNumber(-rv)
	}

	return HostStatusFlag(host.flags & C.SANLK_HOST_MASK), nil
}

func AcquireResourceLease(resources []string, owner string) error {
	if len(resources) > C.SANLK_MAX_RESOURCES {
		return fmt.Errorf("requested resource over max %d", C.SANLK_MAX_RESOURCES)
	}

	sock := C.sanlock_register()
	if sock < 0 {
		return fmt.Errorf("failed to registr process")
	}

	if rv := C.sanlock_restrict(sock, C.SANLK_RESTRICT_SIGTERM); rv < 0 {
		return fmt.Errorf("restrict SIGTERM signal: %s", ErrorNumber(-rv))
	}

	var res_args **C.struct_sanlk_resource
	var res_count C.int
	res := C.CString(strings.Join(resources, " "))
	defer C.free(unsafe.Pointer(res))
	if rv := C.sanlock_state_to_args(res, &res_count, &res_args); rv < 0 {
		return fmt.Errorf("convert sanlock resources: %s", ErrorNumber(-rv))
	}
	opt := C.struct_sanlk_options{
		owner_name: buildSanlockName(owner),
	}

	if rv := C.sanlock_acquire(sock, -1, 0, res_count, res_args, &opt); rv < 0 {
		return fmt.Errorf("acquire resource lease: %s", ErrorNumber(-rv))
	}
	return nil
}

func buildSanlockLockspace(lockspace string, path string, id uint64) C.struct_sanlk_lockspace {
	diskPath := buildSanlockPath(path)
	lockspaceName := buildSanlockName(lockspace)

	disk := C.struct_sanlk_disk{
		path:   diskPath,
		offset: OffsetLockspace,
	}
	return C.struct_sanlk_lockspace{
		name:         lockspaceName,
		host_id:      C.ulong(id),
		host_id_disk: disk,
		flags:        C.SANLK_LSF_ALIGN1M | C.SANLK_LSF_SECTOR512,
	}
}

func buildSanlockPath(path string) [C.SANLK_PATH_LEN]C.char {
	cPath := [C.SANLK_PATH_LEN]C.char{}
	for i := 0; i < len(path); i++ {
		cPath[i] = C.char(path[i])
	}
	return cPath
}

func buildSanlockName(name string) [C.SANLK_NAME_LEN]C.char {
	cName := [C.SANLK_NAME_LEN]C.char{}
	for i := 0; i < len(name); i++ {
		cName[i] = C.char(name[i])
	}
	return cName
}

func buildSanlockRIndex(lockspace string, path string) C.struct_sanlk_rindex {
	diskPath := buildSanlockPath(path)
	lockspaceName := buildSanlockName(lockspace)

	disk := C.struct_sanlk_disk{
		path:   diskPath,
		offset: OffsetRIndex,
	}
	return C.struct_sanlk_rindex{
		flags:          C.SANLK_RIF_ALIGN1M | C.SANLK_RIF_SECTOR512,
		lockspace_name: lockspaceName,
		disk:           disk,
	}
}

func buildSanlockREntry(rEntry string) C.struct_sanlk_rentry {
	rEntryName := buildSanlockName(rEntry)

	return C.struct_sanlk_rentry{
		name: rEntryName,
	}
}
