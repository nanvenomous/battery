package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	path_capacity           = "/sys/class/power_supply/BAT1/capacity"
	path_status             = "/sys/class/power_supply/BAT1/status"
	path_charge_full        = "/sys/class/power_supply/BAT1/charge_full"
	path_charge_full_design = "/sys/class/power_supply/BAT1/charge_full_design"
	path_charge_now         = "/sys/class/power_supply/BAT1/charge_now"
	path_cycle_count        = "/sys/class/power_supply/BAT1/cycle_count"
	path_current_now        = "/sys/class/power_supply/BAT1/current_now"
	path_voltage_now        = "/sys/class/power_supply/BAT1/voltage_now"
	path_manufacturer       = "/sys/class/power_supply/BAT1/manufacturer"
	path_technology         = "/sys/class/power_supply/BAT1/technology"
	path_ac_online          = "/sys/class/power_supply/ACAD/online"
)

func main() {
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

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

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

	// Read additional metrics
	chargeNow, err := intFromFile(path_charge_now)
	if err != nil {
		return err
	}

	currentNow, err := intFromFile(path_current_now)
	if err != nil {
		return err
	}

	voltageNow, err := intFromFile(path_voltage_now)
	if err != nil {
		return err
	}

	manufacturer, err := stringFromFile(path_manufacturer)
	if err != nil {
		manufacturer = "Unknown"
	}

	technology, err := stringFromFile(path_technology)
	if err != nil {
		technology = "Unknown"
	}

	acOnline, err := intFromFile(path_ac_online)
	if err != nil {
		acOnline = 0
	}

	// Calculate power consumption in Watts
	// Power (W) = Current (μA) × Voltage (μV) / 1,000,000,000,000
	powerWatts := float64(currentNow) * float64(voltageNow) / 1000000000000.0

	// Calculate time remaining
	var timeRemaining string
	if currentNow > 0 {
		var hoursRemaining float64
		if acOnline == 1 {
			// Charging: time to full
			hoursRemaining = float64(chargeFull-chargeNow) / float64(currentNow)
			hours := int(hoursRemaining)
			minutes := int((hoursRemaining - float64(hours)) * 60)
			timeRemaining = fmt.Sprintf("Full in %dh %dm", hours, minutes)
		} else {
			// Discharging: time until empty
			hoursRemaining = float64(chargeNow) / float64(currentNow)
			hours := int(hoursRemaining)
			minutes := int((hoursRemaining - float64(hours)) * 60)
			timeRemaining = fmt.Sprintf("%dh %dm left", hours, minutes)
		}
	} else {
		timeRemaining = "N/A"
	}

	// Try to read temperature (may not be available on all systems)
	temperature := "N/A"
	// Temperature sensors vary by system, skip for now if not found

	columns := []table.Column{
		{Title: " ", Width: 1},
		{Title: "Stat", Width: 15},
		{Title: "Value", Width: 20},
	}

	// Format power consumption display
	var powerDisplay string
	if acOnline == 1 {
		powerDisplay = fmt.Sprintf("%.2fW (Charging)", powerWatts)
	} else if powerWatts > 0 {
		powerDisplay = fmt.Sprintf("%.2fW (Draining)", powerWatts)
	} else {
		powerDisplay = "0W (Idle)"
	}

	rows := []table.Row{
		{"󰁹", "Capacity", fmt.Sprintf("%s%%", batteryCapacity)},
		{"󱖫", "Status", fmt.Sprintf("%s", batteryStatus)},
		{"󰚥", "Power", powerDisplay},
		{"⏱", "Time", timeRemaining},
		{"󱈑", "Health", fmt.Sprintf("%.2f%%", batteryHealth*100)},
		{"⭘", "Cycle Count", fmt.Sprintf("%d", cycleCount)},
		{"󰈏", "Manufacturer", manufacturer},
		{"", "Technology", technology},
		{"", "Temperature", temperature},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Bold(false)
	t.SetStyles(s)

	// Parse capacity as int
	capacityInt, err := strconv.Atoi(batteryCapacity)
	if err != nil {
		return err
	}

	// Create progress bar
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	m := model{
		table:    t,
		progress: prog,
		capacity: capacityInt,
		acOnline: acOnline == 1,
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return nil
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table    table.Model
	progress progress.Model
	capacity int
	acOnline bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// Create battery terminal style progress bar
	percent := float64(m.capacity) / 100.0
	progressBar := m.progress.ViewAs(percent)

	// AC power indicator
	var acIndicator string
	if m.acOnline {
		acIndicator = " ⚡ AC"
	} else {
		acIndicator = ""
	}

	// Build battery terminal display with box drawing characters
	batteryDisplay := fmt.Sprintf("  ┌──────────────────────────────────────────┐ ┃\n")
	batteryDisplay += fmt.Sprintf("──┤ %s ├─┃  %d%%%s\n", progressBar, m.capacity, acIndicator)
	batteryDisplay += fmt.Sprintf("  └──────────────────────────────────────────┘ ┃\n")

	return batteryDisplay + "\n" + baseStyle.Render(m.table.View()) + "\n"
}
