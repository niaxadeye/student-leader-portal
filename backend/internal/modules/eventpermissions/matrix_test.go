package eventpermissions

import (
	"slices"
	"testing"
)

func TestPermissionMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		required string
		grants   []string
	}{
		{ParticipantsManage, []string{ParticipantsManage}},
		{AttendanceScan, []string{AttendanceScan, AttendanceManage}},
		{AttendanceManage, []string{AttendanceManage}},
		{TasksModerate, []string{TasksModerate, TasksManage}},
		{TasksManage, []string{TasksManage}},
		{MerchManage, []string{MerchManage}},
		{MerchOrdersManage, []string{MerchOrdersManage}},
		{PointsManage, []string{PointsManage}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.required, func(t *testing.T) {
			t.Parallel()
			if got := GrantsFor(test.required); !slices.Equal(got, test.grants) {
				t.Fatalf("GrantsFor(%q)=%v, want %v", test.required, got, test.grants)
			}
		})
	}
}

func TestPermissionMatrixRejectsUnknownPermission(t *testing.T) {
	t.Parallel()
	if IsKnown("event.unknown.manage") || GrantsFor("event.unknown.manage") != nil {
		t.Fatal("unknown permission must not be granted")
	}
}
