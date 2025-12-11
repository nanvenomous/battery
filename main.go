package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// FULL=$(cat /sys/class/power_supply/BAT1/charge_full)
// DESIGN=$(cat /sys/class/power_supply/BAT1/charge_full_design)
// HEALTH=$(awk "BEGIN {printf \"%.1f\", $FULL/$DESIGN*100}")
// CYCLES=$(cat /sys/class/power_supply/BAT1/cycle_count)
// echo "🔋 ${HEALTH}% health (${CYCLES} cycles)"

const (
	path_capacity           = "/sys/class/power_supply/BAT1/capacity"
	path_status             = "/sys/class/power_supply/BAT1/status"
	path_charge_full        = "/sys/class/power_supply/BAT1/charge_full"
	path_charge_full_design = "/sys/class/power_supply/BAT1/charge_full_design"
	path_cycle_count        = "/sys/class/power_supply/BAT1/cycle_count"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	err := battery()
	if err != nil {
		panic(err)
	}
}

func stringFromFile(filePth string) (string, error) {
	chargeFullByt, err := os.ReadFile(filePth)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(chargeFullByt)), nil
}

func intFromFile(filePth string) (int, error) {
	str, err := stringFromFile(filePth)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(str)
}

func battery() error {
	batteryCapacity, err := stringFromFile(path_capacity)
	if err != nil {
		return err
	}

	batteryStatus, err := stringFromFile(path_status)
	if err != nil {
		return err
	}

	chargeFull, err := intFromFile(path_charge_full)
	if err != nil {
		return err
	}

	chargeFullDesign, err := intFromFile(path_charge_full_design)
	if err != nil {
		return err
	}

	batteryHealth := float32(chargeFull) / float32(chargeFullDesign)

	cycleCount, err := intFromFile(path_cycle_count)
	if err != nil {
		return err
	}

	log.Printf("󰁹 Capacity: %s%%\n", batteryCapacity)
	log.Printf("󱖫 Status: %s\n", batteryStatus)
	log.Printf("󱈑 Health: %.2f\n", batteryHealth*100)
	log.Printf(" Cycle Count: %d\n", cycleCount)

	return nil
}
