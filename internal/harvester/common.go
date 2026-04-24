package harvester

import "k8s.io/apimachinery/pkg/api/resource"

// HandleCPUOverCommitment calculates the CPU Request based on the CPU Limit and the CPU Overcommitment setting.
func HandleCPUOverCommitment(overCommitSettingMap map[string]int, cpuNumber int64) resource.Quantity {
	// cpuQuantity := resource.NewQuantity(cpuNumber, resource.DecimalSI)
	cpuOvercommit := overCommitSettingMap["cpu"]
	if cpuOvercommit <= 0 {
		cpuOvercommit = 100 // default value
	}

	cpuRequest := (1000 * cpuNumber) * 100 / int64(cpuOvercommit)

	return *resource.NewMilliQuantity(cpuRequest, resource.DecimalSI)
}

// HandleMemoryOverCommitment calculates the memory Request based on the memory Limit and the memory Overcommitment setting.
func HandleMemoryOverCommitment(overCommitSettingMap map[string]int, memory string) resource.Quantity {
	// cpuQuantity := resource.NewQuantity(cpuNumber, resource.DecimalSI)
	memoryRequest := resource.MustParse(memory)
	memoryValue := memoryRequest.Value()

	memOvercommit := overCommitSettingMap["memory"]
	if memOvercommit <= 0 {
		memOvercommit = 100 // default value
	}

	return *resource.NewQuantity(memoryValue*100/int64(memOvercommit), resource.BinarySI)
}
