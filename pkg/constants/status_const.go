package constants

const StatusInactive int = 0
const StatusActive int = 1

func GetStatuses() []int {
	return []int{
		StatusInactive,
		StatusActive,
	}
}
