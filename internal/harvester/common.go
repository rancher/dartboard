package harvester

import "k8s.io/apimachinery/pkg/api/resource"

// HandleCPUOverCommitment calculates the CPU Request based on the CPU Limit and the CPU Overcommitment setting.
func HandleCPUOverCommitment(overCommitSettingMap map[string]int, cpuNumber int64) resource.Quantity {
	//cpuQuantiy := resource.NewQuantity(cpuNumber, resource.DecimalSI)
	cpuRequest := (1000 * cpuNumber) * 100 / int64(overCommitSettingMap["cpu"])
	return *resource.NewMilliQuantity(cpuRequest, resource.DecimalSI)
}

// HandleMemoryOverCommitment calculates the memory Request based on the memory Limit and the memory Overcommitment setting.
func HandleMemoryOverCommitment(overCommitSettingMap map[string]int, memory string) resource.Quantity {
	//cpuQuantiy := resource.NewQuantity(cpuNumber, resource.DecimalSI)
	memoryRequest := resource.MustParse(memory)
	memoryValue := memoryRequest.Value()

	return *resource.NewQuantity(memoryValue*100/int64(overCommitSettingMap["memory"]), resource.BinarySI)
}
