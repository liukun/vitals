//go:build darwin

package core

import (
	"context"
	"os/exec"

	"howett.net/plist"
)

func batteryInfo(ctx context.Context) (BatteryInfo, bool) {
	output, err := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-rn", "AppleSmartBattery", "-a").Output()
	if err != nil {
		return BatteryInfo{}, false
	}
	var roots []map[string]any
	if _, err := plist.Unmarshal(output, &roots); err != nil || len(roots) == 0 {
		return BatteryInfo{}, false
	}
	return batteryInfoFromProperties(roots[0]), true
}

func batteryInfoFromProperties(properties map[string]any) BatteryInfo {
	values := make(map[string]any)
	flattenBatteryValues(properties, values)

	charge := intValue(values["CurrentCapacity"])
	design := intValue(values["DesignCapacity"])
	full := firstPositive(
		intValue(values["FullChargeCapacity"]),
		intValue(values["NominalChargeCapacity"]),
	)
	health := 0
	if design > 0 && full > 0 {
		health = min(100, full*100/design)
	}
	temperature := float64(intValue(values["Temperature"]))
	if temperature > 1000 {
		temperature = temperature/10 - 273.15
	} else if temperature > 100 {
		temperature /= 100
	}
	return BatteryInfo{
		Present:       true,
		ChargePercent: min(100, max(0, charge)),
		HealthPercent: health,
		CycleCount:    max(0, intValue(values["CycleCount"])),
		TemperatureC:  temperature,
		Charging:      boolValue(values["IsCharging"]),
		ExternalPower: boolValue(values["ExternalConnected"]),
	}
}

func flattenBatteryValues(value any, result map[string]any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if _, exists := result[key]; !exists {
				result[key] = child
			}
		}
		for _, child := range value {
			flattenBatteryValues(child, result)
		}
	case []any:
		for _, child := range value {
			flattenBatteryValues(child, result)
		}
	}
}

func intValue(value any) int {
	switch value := value.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case uint64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func boolValue(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case int64:
		return value != 0
	case uint64:
		return value != 0
	default:
		return false
	}
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
