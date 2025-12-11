package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

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

//go:embed version
var version string

type tickMsg time.Time

type batteryData struct {
	capacity      int
	status        string
	powerDisplay  string
	timeRemaining string
	health        string
	cycleCount    int
	manufacturer  string
	technology    string
	temperature   string
	acOnline      bool
}

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

func readBatteryData() batteryData {
	data := batteryData{
		capacity:      0,
		status:        "Unknown",
		powerDisplay:  "N/A",
		timeRemaining: "N/A",
		health:        "N/A",
		cycleCount:    0,
		manufacturer:  "Unknown",
		technology:    "Unknown",
		temperature:   "N/A",
		acOnline:      false,
	}

	batteryCapacity, err := stringFromFile(path_capacity)
	if err != nil {
		return data
	}

	batteryStatus, err := stringFromFile(path_status)
	if err != nil {
		return data
	}
	data.status = batteryStatus

	chargeFull, err := intFromFile(path_charge_full)
	if err != nil {
		return data
	}

	chargeFullDesign, err := intFromFile(path_charge_full_design)
	if err != nil {
		return data
	}

	batteryHealth := float32(chargeFull) / float32(chargeFullDesign)
	data.health = fmt.Sprintf("%.2f%%", batteryHealth*100)

	cycleCount, err := intFromFile(path_cycle_count)
	if err == nil {
		data.cycleCount = cycleCount
	}

	chargeNow, err := intFromFile(path_charge_now)
	if err != nil {
		return data
	}

	currentNow, err := intFromFile(path_current_now)
	if err != nil {
		return data
	}

	voltageNow, err := intFromFile(path_voltage_now)
	if err != nil {
		return data
	}

	manufacturer, err := stringFromFile(path_manufacturer)
	if err == nil {
		data.manufacturer = manufacturer
	}

	technology, err := stringFromFile(path_technology)
	if err == nil {
		data.technology = technology
	}

	acOnline, err := intFromFile(path_ac_online)
	if err == nil {
		data.acOnline = acOnline == 1
	}

	// Calculate power consumption in Watts
	powerWatts := float64(currentNow) * float64(voltageNow) / 1000000000000.0

	// Format power consumption display
	if data.acOnline || powerWatts > 0 {
		data.powerDisplay = fmt.Sprintf("%.2fW", powerWatts)
	} else {
		data.powerDisplay = "0W"
	}

	// Calculate time remaining
	if currentNow > 0 {
		var hoursRemaining float64
		if data.acOnline {
			// Charging: time to full
			hoursRemaining = float64(chargeFull-chargeNow) / float64(currentNow)
			hours := int(hoursRemaining)
			minutes := int((hoursRemaining - float64(hours)) * 60)
			data.timeRemaining = fmt.Sprintf("Full in %dh %dm", hours, minutes)
		} else {
			// Discharging: time until empty
			hoursRemaining = float64(chargeNow) / float64(currentNow)
			hours := int(hoursRemaining)
			minutes := int((hoursRemaining - float64(hours)) * 60)
			data.timeRemaining = fmt.Sprintf("%dh %dm left", hours, minutes)
		}
	}

	// Parse capacity as int
	capacityInt, err := strconv.Atoi(batteryCapacity)
	if err == nil {
		data.capacity = capacityInt
	}

	return data
}

func battery() error {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	var firstArg string = ""

	if len(os.Args) >= 2 {
		firstArg = os.Args[1]
	}

	switch firstArg {
	case "version", "--version", "-v":
		fmt.Printf("%s\n", strings.TrimSpace(version))
		return nil
	}

	// Read initial battery data
	data := readBatteryData()

	// Set up table columns
	columns := []table.Column{
		{Title: " ", Width: 1},
		{Title: "Stat", Width: 15},
		{Title: "Value", Width: 20},
	}

	// Create initial table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
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

	// Create progress bar
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	// Create model
	m := model{
		table:    t,
		progress: prog,
		data:     data,
	}

	// Build initial table
	m.buildTable()

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
	data     batteryData
}

func (m *model) buildTable() {
	rows := []table.Row{
		{"󱖫", "Status", m.data.status},
		{"󰚥", "Power", m.data.powerDisplay},
		{"⏱", "Time", m.data.timeRemaining},
		{"󱈑", "Health", m.data.health},
		{"⭘", "Cycle Count", fmt.Sprintf("%d", m.data.cycleCount)},
		{"󰈏", "Manufacturer", m.data.manufacturer},
		{"", "Technology", m.data.technology},
		{"", "Temperature", m.data.temperature},
	}
	m.table.SetRows(rows)
}

func tickCmd() tea.Msg {
	time.Sleep(time.Second)
	return tickMsg(time.Now())
}

func (m model) Init() tea.Cmd {
	return tickCmd
}

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
	case tickMsg:
		// Update battery data on tick
		m.data = readBatteryData()
		m.buildTable()
		return m, tickCmd
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	// Create battery terminal style progress bar
	percent := float64(m.data.capacity) / 100.0
	progressBar := m.progress.ViewAs(percent)

	// AC power indicator
	var acIndicator string
	if m.data.acOnline {
		acIndicator = " ⚡ AC"
	} else {
		acIndicator = ""
	}

	// Build battery terminal display with box drawing characters
	batteryDisplay := fmt.Sprintf("  ┌──────────────────────────────────────────┐ ┃\n")
	batteryDisplay += fmt.Sprintf("──┤ %s ├─┃  %d%%%s\n", progressBar, m.data.capacity, acIndicator)
	batteryDisplay += fmt.Sprintf("  └──────────────────────────────────────────┘ ┃\n")

	return batteryDisplay + "\n" + baseStyle.Render(m.table.View()) + "\n"
}
