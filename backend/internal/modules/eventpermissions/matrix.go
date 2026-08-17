// Package eventpermissions defines the server-side scoped permission matrix for events.
package eventpermissions

const (
	ParticipantsManage = "event.participants.manage"
	AttendanceScan     = "event.attendance.scan"
	AttendanceManage   = "event.attendance.manage"
	TasksManage        = "event.tasks.manage"
	TasksModerate      = "event.tasks.moderate"
	MerchManage        = "event.merch.manage"
	MerchOrdersManage  = "event.merch.orders.manage"
	PointsManage       = "event.points.manage"
)

// GrantsFor returns the explicit permissions that satisfy required. Higher
// operational permissions imply only the narrower action in the same module.
// Event ownership and MEGA_ADMIN are evaluated outside this matrix.
func GrantsFor(required string) []string {
	switch required {
	case AttendanceScan:
		return []string{AttendanceScan, AttendanceManage}
	case TasksModerate:
		return []string{TasksModerate, TasksManage}
	case ParticipantsManage, AttendanceManage, TasksManage, MerchManage, MerchOrdersManage, PointsManage:
		return []string{required}
	default:
		return nil
	}
}

func IsKnown(permission string) bool {
	return len(GrantsFor(permission)) > 0
}
