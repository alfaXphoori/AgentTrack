package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func runDashboard() {
	loadConfig()

	// Set global tview styles to use terminal default colors (transparency)
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorDefault
	tview.Styles.ContrastBackgroundColor = tcell.ColorDefault
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorDefault

	app := tview.NewApplication()
	var layout tview.Primitive
	restoreDashboard := func() {
		if layout != nil {
			app.SetRoot(layout, true)
		}
	}

	// Header
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprintf("[yellow::b]AgentTrack Dashboard[white] | [green]%s", time.Now().Format("2006-01-02 15:04")))

	// Pages for the tabs
	infoPages := tview.NewPages()

	// Tab bar
	tabBar := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)

	tabOverview, updateOverview := createOverviewTab()
	tabLogs := createLogsTab(app)
	tabStats, updateStats := createStatsTab()
	tabTrends, updateTrends := createTrendsTab()
	tabCost, updateCost := createCostTab()
	tabProjects, updateProjects := createProjectsTab()
	tabHeatmap, updateHeatmap := createHeatmapTab()
	tabSearch := createSearchTab(app)
	tabTags, updateTags := createTagsTab()
	tabSettings := createSettingsTab(app, restoreDashboard)

	infoPages.AddPage("Overview", tabOverview, true, true)
	infoPages.AddPage("Logs", tabLogs, true, false)
	infoPages.AddPage("Stats", tabStats, true, false)
	infoPages.AddPage("Trends", tabTrends, true, false)
	infoPages.AddPage("Heatmap", tabHeatmap, true, false)
	infoPages.AddPage("Cost", tabCost, true, false)
	infoPages.AddPage("Projects", tabProjects, true, false)
	infoPages.AddPage("Search", tabSearch, true, false)
	infoPages.AddPage("Tags", tabTags, true, false)
	infoPages.AddPage("Settings", tabSettings, true, false)

	// Create the tab text
	tabs := []string{"Overview", "Logs", "Stats", "Trends", "Heatmap", "Cost", "Projects", "Tags", "Search", "Settings"}
	updateTabBar := func(activeTab string) {
		tabBar.Clear()
		for i, tab := range tabs {
			if tab == activeTab {
				fmt.Fprintf(tabBar, `["%s"][black:green:b] %d. %s [white:-:-][""]  `, tab, i+1, tab)
			} else {
				fmt.Fprintf(tabBar, `["%s"][white:black:-] %d. %s [white:-:-][""]  `, tab, i+1, tab)
			}
		}
	}

	activeTabName := "Overview"
	updateTabBar("Overview")
	tabBar.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			activeTabName = added[0]
			infoPages.SwitchToPage(added[0])
			updateTabBar(added[0])
		}
	})

	// Global Auto-Refresh Ticker
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				app.QueueUpdateDraw(func() {
					// Refresh header time
					header.SetText(fmt.Sprintf("[yellow::b]AgentTrack Dashboard[white] | [green]%s", time.Now().Format("2006-01-02 15:04")))

					// Refresh active tab data
					switch activeTabName {
					case "Overview":
						updateOverview()
					case "Stats":
						updateStats()
					case "Trends":
						updateTrends()
					case "Heatmap":
						updateHeatmap()
					case "Cost":
						updateCost()
					case "Projects":
						updateProjects()
					case "Tags":
						updateTags()
					}
				})
			}
		}
	}()

	// Footer
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]1-7[white] Tabs | [yellow]/[white] Search | [yellow]↑/↓[white] Navigate | [yellow]q[white] Quit")

	// Global key handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				app.Stop()
				return nil
			case '/':
				tabBar.Highlight("Search")
				return nil
			case '1':
				tabBar.Highlight("Overview")
			case '2':
				tabBar.Highlight("Logs")
			case '3':
				tabBar.Highlight("Stats")
			case '4':
				tabBar.Highlight("Trends")
			case '5':
				tabBar.Highlight("Cost")
			case '6':
				tabBar.Highlight("Search")
			case '7':
				tabBar.Highlight("Settings")
			}
		}
		return event
	})

	layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(tabBar, 1, 0, false).
		AddItem(infoPages, 0, 1, true).
		AddItem(footer, 1, 0, false)

	if err := app.SetRoot(layout, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func createOverviewTab() (tview.Primitive, func()) {
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Mode state
	mode := "Today"
	customDays := 7

	contentArea := tview.NewFlex().SetDirection(tview.FlexRow)

	updateOverview := func() {
		contentArea.Clear()
		logs := getLogsFromAllFiles()

		now := time.Now()
		var cutoff time.Time
		title := ""

		switch mode {
		case "Today":
			cutoff = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			title = "Today's Snapshot"
		case "Week":
			cutoff = now.AddDate(0, 0, -7)
			title = "Last 7 Days Snapshot"
		case "Month":
			cutoff = now.AddDate(0, 0, -30)
			title = "Last 30 Days Snapshot"
		case "Custom":
			cutoff = now.AddDate(0, 0, -customDays)
			title = fmt.Sprintf("Last %d Days Snapshot", customDays)
		}

		var filteredLogs []LogEntry
		tIn, tOut := 0, 0
		cost := 0.0
		models := make(map[string]int)

		for _, l := range logs {
			logTime, err := time.Parse("2006-01-02 15:04:05", l.Timestamp)
			if err == nil && !logTime.Before(cutoff) {
				filteredLogs = append(filteredLogs, l)
				tIn += l.TokensIn
				tOut += l.TokensOut
				if c, ok := calculateLogCost(l); ok {
					cost += c
				}
				m := logModel(l)
				if m != "" {
					models[m]++
				}
			}
		}

		// Snapshot View
		snapshotText := fmt.Sprintf(`[cyan::b]%s[white::-]
Logs Recorded: [green]%d[white]
Total Tokens:  [yellow]%d[white] (In: %d, Out: %d)
Estimated Cost: [green]%s %.4f[white]`, title, len(filteredLogs), tIn+tOut, tIn, tOut, config.Pricing.Currency, cost)

		snapshotView := tview.NewTextView().SetDynamicColors(true).SetText(snapshotText)
		snapshotView.SetBorder(true).SetTitle(" Summary ")

		// Top models
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range models {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

		modelsText := ""
		for i, item := range sorted {
			if i >= 5 {
				break
			}
			modelsText += fmt.Sprintf("[blue]%d.[white] %-20s : [green]%d[white] logs\n", i+1, item.k, item.v)
		}

		modelsView := tview.NewTextView().SetDynamicColors(true).SetText(modelsText)
		modelsView.SetBorder(true).SetTitle(" Top Models ")

		// Recent Activity (Limit to filtered logs)
		recentText := ""
		limit := 10
		if len(filteredLogs) < limit {
			limit = len(filteredLogs)
		}

		for i := len(filteredLogs) - 1; i >= len(filteredLogs)-limit; i-- {
			l := filteredLogs[i]
			msg := l.Message
			if l.Category == "AutoLog" && l.Question != "" {
				msg = l.Question
				if len(msg) > 60 {
					msg = msg[:57] + "..."
				}
			}
			ts := ""
			if len(l.Timestamp) >= 16 {
				ts = l.Timestamp[5:16]
			}
			recentText += fmt.Sprintf("[cyan]%s[white] | [yellow]%-15s[white] | %s\n", ts, logModel(l), msg)
		}

		recentView := tview.NewTextView().SetDynamicColors(true).SetText(recentText)
		recentView.SetBorder(true).SetTitle(" Recent Activity ")

		topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(snapshotView, 0, 1, false).
			AddItem(modelsView, 0, 1, false)

		contentArea.AddItem(topRow, 7, 1, false).
			AddItem(recentView, 0, 2, false)
	}

	// Filter Controls
	filterForm := tview.NewForm().
		AddDropDown("View Period", []string{"Today", "Week", "Month", "Custom"}, 0, func(option string, optionIndex int) {
			mode = option
			updateOverview()
		}).
		AddInputField("Custom Days", "7", 5, nil, func(text string) {
			if val, err := strconv.Atoi(text); err == nil && val > 0 {
				customDays = val
				if mode == "Custom" {
					updateOverview()
				}
			}
		})
	filterForm.SetHorizontal(true).SetBorder(false)

	updateOverview()

	mainFlex.AddItem(filterForm, 3, 0, true).
		AddItem(contentArea, 0, 1, false)

	return mainFlex, updateOverview
}

func createLogsTab(app *tview.Application) tview.Primitive {
	logs := getLogsFromAllFiles()
	var filteredLogs []LogEntry
	var stopCh chan struct{}

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)

	detailView := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetWordWrap(true)
	detailView.SetBorder(true).SetTitle(" Detail ")

	renderDetail := func(l LogEntry) {
		costStr := "n/a"
		if c, ok := calculateLogCost(l); ok {
			costStr = fmt.Sprintf("%s %.4f", config.Pricing.Currency, c)
		}

		detailText := fmt.Sprintf(`[yellow::b]Timestamp:[white::-] %s
[yellow::b]Category:[white::-]  %s
[yellow::b]Model:[white::-]     %s
[yellow::b]Tokens:[white::-]    In: %d | Out: %d | Total: %d
[yellow::b]Time:[white::-]      %.2fs
[yellow::b]Cost:[white::-]      %s
[yellow::b]Tags:[white::-]      %s

`, l.Timestamp, logCategory(l), logModel(l), l.TokensIn, l.TokensOut, l.TokensIn+l.TokensOut, l.Duration, costStr, strings.Join(l.Tags, ", "))

		if l.Category == "AutoLog" {
			detailText += fmt.Sprintf("[cyan::b]Question:[white::-]\n%s\n\n[green::b]Answer:[white::-]\n%s", l.Question, l.Answer)
		} else {
			detailText += fmt.Sprintf("[cyan::b]Message:[white::-]\n%s", l.Message)
		}
		detailView.SetText(detailText)
	}

	keyword := ""
	model := ""
	category := ""
	updateTable := func(reload bool) {
		if reload {
			logs = getLogsFromAllFiles()
		}

		table.Clear()
		headers := []string{"Time", "Model", "Category", "Tokens", "Time", "Content"}
		for i, h := range headers {
			table.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false).SetExpansion(1))
		}

		rawFiltered := filterLogs(logs, FilterOptions{Keyword: keyword, Model: model, Category: category})
		filteredLogs = filteredLogs[:0]
		row := 1
		for i := len(rawFiltered) - 1; i >= 0; i-- {
			l := rawFiltered[i]
			filteredLogs = append(filteredLogs, l)

			msg := l.Message
			if l.Category == "AutoLog" && l.Question != "" {
				msg = "Q: " + l.Question
			}
			if len(msg) > 80 {
				msg = msg[:77] + "..."
			}

			tokens := fmt.Sprintf("%d", l.TokensIn+l.TokensOut)
			duration := ""
			if l.Duration > 0 {
				duration = fmt.Sprintf("%.2fs", l.Duration)
			}

			timestamp := l.Timestamp
			if len(timestamp) >= 16 {
				timestamp = timestamp[5:16]
			}

			table.SetCell(row, 0, tview.NewTableCell(timestamp).SetTextColor(tcell.ColorTeal))
			table.SetCell(row, 1, tview.NewTableCell(logModel(l)).SetTextColor(tcell.ColorBlue))
			table.SetCell(row, 2, tview.NewTableCell(logCategory(l)).SetTextColor(tcell.ColorPurple))
			table.SetCell(row, 3, tview.NewTableCell(tokens).SetTextColor(tcell.ColorYellow))
			table.SetCell(row, 4, tview.NewTableCell(duration).SetTextColor(tcell.ColorGreen))
			table.SetCell(row, 5, tview.NewTableCell(msg).SetTextColor(tcell.ColorWhite))
			row++
		}

		if len(filteredLogs) > 0 {
			table.Select(1, 0)
			renderDetail(filteredLogs[0])
		} else {
			detailView.SetText("")
		}
	}

	stopLive := func() {
		if stopCh != nil {
			close(stopCh)
			stopCh = nil
		}
	}

	startLive := func() {
		stopLive()
		currentStop := make(chan struct{})
		stopCh = currentStop
		go func(stop <-chan struct{}) {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					app.QueueUpdateDraw(func() {
						if stopCh != currentStop {
							return
						}
						updateTable(true)
					})
				case <-stop:
					return
				}
			}
		}(currentStop)
	}

	filterForm := tview.NewForm().
		AddInputField("Keyword", "", 20, nil, func(text string) {
			keyword = text
			updateTable(true)
		}).
		AddInputField("Model", "", 20, nil, func(text string) {
			model = text
			updateTable(true)
		}).
		AddInputField("Category", "", 15, nil, func(text string) {
			category = text
			updateTable(true)
		}).
		AddCheckbox("Live", false, func(checked bool) {
			if checked {
				startLive()
			} else {
				stopLive()
			}
			updateTable(true)
		})
	filterForm.SetHorizontal(true).SetBorder(false)

	table.SetSelectionChangedFunc(func(row, column int) {
		if row <= 0 || row > len(filteredLogs) {
			detailView.SetText("")
			return
		}
		renderDetail(filteredLogs[row-1])
	})

	updateTable(true)

	contentFlex := tview.NewFlex().
		AddItem(table, 0, 2, true).
		AddItem(detailView, 0, 1, false)

	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filterForm, 3, 0, false).
		AddItem(contentFlex, 0, 1, true)
}

func createStatsTab() (tview.Primitive, func()) {
	var rows []ModelStats
	// Bar chart metric selector
	metric := "Logs" // default
	chartView := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	chartView.SetBorder(true).SetTitle(" Bar Chart ")

	barColors := []string{"green", "yellow", "blue", "red", "cyan", "purple", "white", "teal"}

	renderChart := func() {
		logs := getLogsFromAllFiles()
		rows = collectModelStats(logs)
		chartView.Clear()
		if len(rows) == 0 {
			fmt.Fprintf(chartView, "\n  [gray]No data available.[white]")
			return
		}

		// Find max value for scaling
		maxVal := 0
		for _, r := range rows {
			var v int
			switch metric {
			case "Logs":
				v = r.Logs
			case "Tokens In":
				v = r.TokensIn
			case "Tokens Out":
				v = r.TokensOut
			case "Total Tokens":
				v = r.TokensIn + r.TokensOut
			}
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}

		const barWidth = 40
		fmt.Fprintf(chartView, "\n  [yellow::b]%-28s  %-*s  Value[white::-]\n\n", "Model", barWidth, "")

		for i, r := range rows {
			var val int
			switch metric {
			case "Logs":
				val = r.Logs
			case "Tokens In":
				val = r.TokensIn
			case "Tokens Out":
				val = r.TokensOut
			case "Total Tokens":
				val = r.TokensIn + r.TokensOut
			}

			filled := int(float64(val) / float64(maxVal) * barWidth)
			if filled == 0 && val > 0 {
				filled = 1
			}

			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			color := barColors[i%len(barColors)]

			modelName := r.Model
			if len(modelName) > 27 {
				modelName = modelName[:24] + "..."
			}

			extra := ""
			if metric == "Total Tokens" && r.HasCost {
				extra = fmt.Sprintf("  [green](%s %.4f)[white]", config.Pricing.Currency, r.Cost)
			}

			fmt.Fprintf(chartView, "  [white]%-28s[white]  [%s]%s[white]  [cyan]%d[white]%s\n\n", modelName, color, bar, val, extra)
		}
	}

	// Metric selector
	metricForm := tview.NewForm().
		AddDropDown("Metric", []string{"Logs", "Tokens In", "Tokens Out", "Total Tokens"}, 0, func(option string, _ int) {
			metric = option
			renderChart()
		})
	metricForm.SetHorizontal(true).SetBorder(false)

	renderChart()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(metricForm, 3, 0, true).
		AddItem(chartView, 0, 1, false)

	return flex, renderChart
}

func createTrendsTab() (tview.Primitive, func()) {
	metric := "Logs"
	chartView := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	chartView.SetBorder(true).SetTitle(" Daily Trends ")

	type trendRow struct {
		Day    time.Time
		Logs   int
		Tokens int
		Cost   float64
	}

	valueForMetric := func(row trendRow) float64 {
		switch metric {
		case "Tokens":
			return float64(row.Tokens)
		case "Cost":
			return row.Cost
		default:
			return float64(row.Logs)
		}
	}

	renderChart := func() {
		logs := getLogsFromAllFiles()
		chartView.Clear()

		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -29)
		daily := make(map[string]*trendRow)

		for _, log := range logs {
			logTime, err := parseTimestamp(log.Timestamp)
			if err != nil {
				continue
			}

			day := time.Date(logTime.Year(), logTime.Month(), logTime.Day(), 0, 0, 0, 0, logTime.Location())
			if day.Before(start) || day.After(now) {
				continue
			}

			key := day.Format("2006-01-02")
			if daily[key] == nil {
				daily[key] = &trendRow{Day: day}
			}

			row := daily[key]
			row.Logs++
			row.Tokens += log.TokensIn + log.TokensOut
			if cost, ok := calculateLogCost(log); ok {
				row.Cost += cost
			}
		}

		var rows []trendRow
		maxVal := 0.0
		for _, row := range daily {
			val := valueForMetric(*row)
			if val <= 0 {
				continue
			}
			rows = append(rows, *row)
			if val > maxVal {
				maxVal = val
			}
		}

		if len(rows) == 0 {
			fmt.Fprintf(chartView, "\n  [gray]No data available[white]")
			return
		}

		sort.Slice(rows, func(i, j int) bool {
			return rows[i].Day.Before(rows[j].Day)
		})
		if maxVal == 0 {
			maxVal = 1
		}

		barColor := "green"
		if metric == "Tokens" {
			barColor = "yellow"
		} else if metric == "Cost" {
			barColor = "cyan"
		}

		fmt.Fprintf(chartView, "\n  [yellow::b]%-10s %-40s %s[white::-]\n\n", "Date", "Bar", "Value")
		for _, row := range rows {
			val := valueForMetric(row)
			filled := int((val / maxVal) * 40)
			if filled == 0 && val > 0 {
				filled = 1
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", 40-filled)

			valueText := fmt.Sprintf("%d", int(val))
			if metric == "Cost" {
				valueText = fmt.Sprintf("%s %.4f", config.Pricing.Currency, val)
			}

			fmt.Fprintf(chartView, "  [white]%-10s[white] [%s]%s[white] %s\n", row.Day.Format("01-02"), barColor, bar, valueText)
		}
	}

	metricForm := tview.NewForm().
		AddDropDown("Metric", []string{"Logs", "Tokens", "Cost"}, 0, func(option string, _ int) {
			metric = option
			renderChart()
		})
	metricForm.SetHorizontal(true).SetBorder(false)

	renderChart()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(metricForm, 3, 0, true).
		AddItem(chartView, 0, 1, false)

	return flex, renderChart
}

func createHeatmapTab() (tview.Primitive, func()) {
	metric := "Logs"
	heatmapView := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	heatmapView.SetBorder(true).SetTitle(" AI Activity Heatmap (Last 20 Weeks) ")

	renderHeatmap := func() {
		logs := getLogsFromAllFiles()
		heatmapView.Clear()

		now := time.Now()
		// Get to the most recent Saturday
		daysToSub := int(now.Weekday()) - int(time.Saturday)
		if daysToSub > 0 {
			daysToSub -= 7
		}
		endOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -daysToSub)
		
		numWeeks := 20
		startDay := endOfWeek.AddDate(0, 0, -(numWeeks*7 - 1)) // 20 weeks ago, Sunday

		dailyVals := make(map[string]int)

		for _, log := range logs {
			logTime, err := parseTimestamp(log.Timestamp)
			if err != nil {
				continue
			}

			day := time.Date(logTime.Year(), logTime.Month(), logTime.Day(), 0, 0, 0, 0, logTime.Location())
			if day.Before(startDay) || day.After(now) {
				continue
			}

			key := day.Format("2006-01-02")
			if metric == "Tokens" {
				dailyVals[key] += log.TokensIn + log.TokensOut
			} else {
				dailyVals[key]++
			}
		}

		getColor := func(val int) string {
			if val == 0 {
				return "[#303030]"
			}
			if metric == "Tokens" {
				if val <= 5000 { return "[#0e4429]" }
				if val <= 20000 { return "[#006d32]" }
				if val <= 50000 { return "[#26a641]" }
				return "[#39d353]"
			} else {
				if val <= 2 { return "[#0e4429]" }
				if val <= 5 { return "[#006d32]" }
				if val <= 10 { return "[#26a641]" }
				return "[#39d353]"
			}
		}

		dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		grid := make([]string, 7)
		
		for w := 0; w < numWeeks; w++ {
			for d := 0; d < 7; d++ {
				currentDay := startDay.AddDate(0, 0, w*7+d)
				if currentDay.After(now) {
					grid[d] += "  "
					continue
				}
				key := currentDay.Format("2006-01-02")
				val := dailyVals[key]
				color := getColor(val)
				grid[d] += fmt.Sprintf("%s■[-] ", color)
			}
		}

		fmt.Fprintln(heatmapView, "")
		for i := 0; i < 7; i++ {
			fmt.Fprintf(heatmapView, "  [white]%s[-]  %s\n", dayNames[i], grid[i])
		}
		
		fmt.Fprintln(heatmapView, "")
		if metric == "Tokens" {
			fmt.Fprintf(heatmapView, "  [gray]Legend: [#303030]■[-] 0  [#0e4429]■[-] 1-5k  [#006d32]■[-] 5k-20k  [#26a641]■[-] 20k-50k  [#39d353]■[-] 50k+ Tokens\n")
		} else {
			fmt.Fprintf(heatmapView, "  [gray]Legend: [#303030]■[-] 0  [#0e4429]■[-] 1-2   [#006d32]■[-] 3-5    [#26a641]■[-] 6-10    [#39d353]■[-] 11+ Logs\n")
		}
	}

	metricForm := tview.NewForm().
		AddDropDown("Metric", []string{"Logs", "Tokens"}, 0, func(option string, _ int) {
			metric = option
			renderHeatmap()
		})
	metricForm.SetHorizontal(true).SetBorder(false)

	renderHeatmap()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(metricForm, 3, 0, true).
		AddItem(heatmapView, 0, 1, false)

	return flex, renderHeatmap
}

func createCostTab() (tview.Primitive, func()) {
	mainFlex := tview.NewFlex()

	summaryView := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	summaryView.SetBorder(true).SetTitle(" Summary ")

	costTable := tview.NewTable().SetBorders(true)
	costTable.SetBorder(true).SetTitle(" Cost Breakdown ")

	updateCost := func() {
		logs := getLogsFromAllFiles()
		rows := collectModelStats(logs)
		summaryView.Clear()
		costTable.Clear()

		formatCost := func(amount float64, ok bool) string {
			if !ok {
				return "n/a"
			}
			return fmt.Sprintf("%s %.4f", config.Pricing.Currency, amount)
		}

		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		var todayCost, weekCost, monthCost, totalCost float64
		var hasToday, hasWeek, hasMonth, hasTotal bool

		for _, log := range logs {
			cost, ok := calculateLogCost(log)
			if !ok {
				continue
			}

			totalCost += cost
			hasTotal = true

			logTime, err := parseTimestamp(log.Timestamp)
			if err != nil {
				continue
			}

			if !logTime.Before(startOfDay) {
				todayCost += cost
				hasToday = true
			}

			age := time.Since(logTime)
			if age >= 0 && age <= 7*24*time.Hour {
				weekCost += cost
				hasWeek = true
			}
			if age >= 0 && age <= 30*24*time.Hour {
				monthCost += cost
				hasMonth = true
			}
		}

		estimatedMonthlyCost := monthCost / 30.0 * 30.0
		summaryText := fmt.Sprintf(`[yellow::b]Today's cost:[white::-] %s
[yellow::b]This week's cost:[white::-] %s
[yellow::b]This month's cost:[white::-] %s
[yellow::b]Total all-time cost:[white::-] %s
[yellow::b]Estimated monthly cost:[white::-] %s`,
			formatCost(todayCost, hasToday),
			formatCost(weekCost, hasWeek),
			formatCost(monthCost, hasMonth),
			formatCost(totalCost, hasTotal),
			formatCost(estimatedMonthlyCost, hasMonth),
		)
		summaryView.SetText(summaryText)

		sort.Slice(rows, func(i, j int) bool {
			if rows[i].HasCost != rows[j].HasCost {
				return rows[i].HasCost
			}
			if rows[i].Cost == rows[j].Cost {
				return strings.ToLower(rows[i].Model) < strings.ToLower(rows[j].Model)
			}
			return rows[i].Cost > rows[j].Cost
		})

		headers := []string{"Model", "Logs", "Total Tokens", "Cost"}
		for i, h := range headers {
			costTable.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false).SetExpansion(1))
		}

		for rowIdx, row := range rows {
			tokens := row.TokensIn + row.TokensOut
			costText := "n/a"
			if row.HasCost {
				costText = fmt.Sprintf("%s %.4f", config.Pricing.Currency, row.Cost)
			}

			costTable.SetCell(rowIdx+1, 0, tview.NewTableCell(row.Model).SetTextColor(tcell.ColorBlue))
			costTable.SetCell(rowIdx+1, 1, tview.NewTableCell(fmt.Sprintf("%d", row.Logs)).SetTextColor(tcell.ColorWhite))
			costTable.SetCell(rowIdx+1, 2, tview.NewTableCell(fmt.Sprintf("%d", tokens)).SetTextColor(tcell.ColorYellow))
			costTable.SetCell(rowIdx+1, 3, tview.NewTableCell(costText).SetTextColor(tcell.ColorGreen))
		}
	}

	updateCost()
	mainFlex.AddItem(summaryView, 0, 1, false).
		AddItem(costTable, 0, 2, true)

	return mainFlex, updateCost
}

func createSearchTab(app *tview.Application) tview.Primitive {
	logs := getLogsFromAllFiles()

	table := tview.NewTable().SetBorders(false).SetSelectable(true, false)
	detailView := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetWordWrap(true)
	detailView.SetBorder(true).SetTitle(" Detail ")

	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldBackgroundColor(tcell.ColorDarkSlateGray).
		SetLabelColor(tcell.ColorYellow)

	// Keep track of filtered logs for selection mapping
	var filteredLogs []LogEntry

	updateTable := func(query string) {
		table.Clear()
		filteredLogs = []LogEntry{}
		headers := []string{"Time", "Model", "Content"}
		for i, h := range headers {
			table.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false))
		}

		row := 1
		query = strings.ToLower(query)
		for i := len(logs) - 1; i >= 0; i-- {
			l := logs[i]
			content := l.Message
			if l.Category == "AutoLog" {
				content = l.Question + " " + l.Answer
			}

			if query != "" && !strings.Contains(strings.ToLower(content), query) && !strings.Contains(strings.ToLower(logModel(l)), query) {
				continue
			}

			filteredLogs = append(filteredLogs, l)

			displayMsg := l.Message
			if l.Category == "AutoLog" {
				displayMsg = "Q: " + l.Question
			}
			if len(displayMsg) > 100 {
				displayMsg = displayMsg[:97] + "..."
			}

			table.SetCell(row, 0, tview.NewTableCell(l.Timestamp[5:16]).SetTextColor(tcell.ColorTeal))
			table.SetCell(row, 1, tview.NewTableCell(logModel(l)).SetTextColor(tcell.ColorBlue))
			table.SetCell(row, 2, tview.NewTableCell(displayMsg).SetTextColor(tcell.ColorWhite))
			row++
		}

		if len(filteredLogs) > 0 {
			table.Select(1, 0)
		} else {
			detailView.SetText("")
		}
	}

	table.SetSelectionChangedFunc(func(row, column int) {
		if row <= 0 || row > len(filteredLogs) {
			detailView.SetText("")
			return
		}

		l := filteredLogs[row-1]
		costStr := "n/a"
		if c, ok := calculateLogCost(l); ok {
			costStr = fmt.Sprintf("%s %.4f", config.Pricing.Currency, c)
		}

		detailText := fmt.Sprintf(`[yellow::b]Timestamp:[white::-] %s
[yellow::b]Category:[white::-]  %s
[yellow::b]Model:[white::-]     %s
[yellow::b]Tokens:[white::-]    In: %d | Out: %d | Total: %d
[yellow::b]Time:[white::-]      %.2fs
[yellow::b]Cost:[white::-]      %s
[yellow::b]Tags:[white::-]      %s

`, l.Timestamp, logCategory(l), logModel(l), l.TokensIn, l.TokensOut, l.TokensIn+l.TokensOut, l.Duration, costStr, strings.Join(l.Tags, ", "))

		if l.Category == "AutoLog" {
			detailText += fmt.Sprintf("[cyan::b]Question:[white::-]\n%s\n\n[green::b]Answer:[white::-]\n%s", l.Question, l.Answer)
		} else {
			detailText += fmt.Sprintf("[cyan::b]Message:[white::-]\n%s", l.Message)
		}
		detailView.SetText(detailText)
	})

	input.SetChangedFunc(updateTable)
	updateTable("")

	// Key handler for Search tab to switch focus
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 1, 0, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(table, 0, 2, false).
			AddItem(detailView, 0, 1, false), 0, 1, false)

	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if input.HasFocus() {
				app.SetFocus(table)
			} else {
				app.SetFocus(input)
			}
			return nil
		}
		return event
	})

	return mainFlex
}

func createSettingsTab(app *tview.Application, restoreDashboard func()) tview.Primitive {
	form := tview.NewForm()

	form.SetBorder(true).SetTitle(" Application Settings ").SetTitleAlign(tview.AlignLeft)

	// Calculate total log size
	var totalSize int64
	files := getAllLogFiles()
	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			totalSize += info.Size()
		}
	}
	sizeStr := fmt.Sprintf("%.2f KB", float64(totalSize)/1024.0)
	if totalSize > 1024*1024 {
		sizeStr = fmt.Sprintf("%.2f MB", float64(totalSize)/(1024.0*1024.0))
	}

	// Export buttons at the top
	form.AddButton("Export MD", func() {
		exportLogs("md")
		showModal(app, restoreDashboard, "Export", "Logs exported to Markdown file.")
	})
	form.AddButton("Export CSV", func() {
		exportLogs("csv")
		showModal(app, restoreDashboard, "Export", "Logs exported to CSV file.")
	})
	form.AddButton("Export JSON", func() {
		exportLogs("json")
		showModal(app, restoreDashboard, "Export", "Logs exported to JSON file.")
	})

	form.AddInputField("Project Name", config.ProjectName, 30, nil, func(text string) {
		config.ProjectName = text
	})

	form.AddInputField("Default Model", config.DefaultModel, 30, nil, func(text string) {
		config.DefaultModel = text
	})

	timezoneOptions := ListAllTimezones()
	initialTimezone := 0
	for i, opt := range timezoneOptions {
		if opt == config.Timezone {
			initialTimezone = i
			break
		}
	}
	form.AddDropDown("Timezone", timezoneOptions, initialTimezone, func(option string, optionIndex int) {
		config.Timezone = option
	})

	form.AddCheckbox("Auto Run Service", config.AutoRun, func(checked bool) {
		config.AutoRun = checked
	})

	form.AddCheckbox("Show Workspace Path", config.Display.ShowWorkspace, func(checked bool) {
		config.Display.ShowWorkspace = checked
	})

	form.AddCheckbox("Reverse Logs Order", config.Display.ReverseOrder, func(checked bool) {
		config.Display.ReverseOrder = checked
	})

	form.AddInputField("Max Logs in View", fmt.Sprintf("%d", config.Display.MaxLogsView), 10, nil, func(text string) {
		if val, err := strconv.Atoi(text); err == nil {
			config.Display.MaxLogsView = val
		}
	})

	form.AddCheckbox("Enable Budget Alerts", config.Budget.Enabled, func(checked bool) {
		config.Budget.Enabled = checked
	})

	form.AddInputField("Max Monthly Budget ($)", fmt.Sprintf("%.2f", config.Budget.MaxMonthlyCost), 15, nil, func(text string) {
		if val, err := strconv.ParseFloat(text, 64); err == nil {
			config.Budget.MaxMonthlyCost = val
		}
	})

	form.AddInputField("Alert Threshold (0.0 - 1.0)", fmt.Sprintf("%.2f", config.Budget.AlertThreshold), 15, nil, func(text string) {
		if val, err := strconv.ParseFloat(text, 64); err == nil {
			config.Budget.AlertThreshold = val
		}
	})

	form.AddInputField("Waste Threshold (Tokens/Req)", fmt.Sprintf("%d", config.Budget.WasteThresholdTokens), 15, nil, func(text string) {
		if val, err := strconv.Atoi(text); err == nil {
			config.Budget.WasteThresholdTokens = val
		}
	})

	form.AddButton("Save Changes", func() {
		if err := saveConfig(); err == nil {
			showModal(app, restoreDashboard, "Success", "Configuration saved successfully!")
		} else {
			showModal(app, restoreDashboard, "Error", fmt.Sprintf("Failed to save config: %v", err))
		}
	})

	form.AddButton("Clear All Logs", func() {
		confirmModal := tview.NewModal().
			SetText("Are you sure you want to clear ALL logs? This cannot be undone.").
			AddButtons([]string{"Clear", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "Clear" {
					clearLogs()
					showModal(app, restoreDashboard, "Success", "All logs have been cleared.")
				} else {
					restoreDashboard()
				}
			})
		app.SetRoot(confirmModal, true)
	})

	form.AddButton("Reset Defaults", func() {
		config = defaultConfig()
		restoreDashboard()
	})

	// Database Storage info at the bottom
	form.AddTextView("Database Storage", fmt.Sprintf("%s (%d files)", sizeStr, len(files)), 0, 1, false, false)

	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).
		AddItem(form, 0, 1, true).
		AddItem(nil, 1, 0, false)
}

func showModal(app *tview.Application, restoreDashboard func(), title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			restoreDashboard()
		})

	app.SetRoot(modal, true)
}

func createProjectsTab() (tview.Primitive, func()) {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)

	currentProject := ""
	var updateFunc func()

	table.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return
		}
		if currentProject == "" {
			cell := table.GetCell(row, 0)
			if cell != nil {
				currentProject = cell.Text
				updateFunc()
			}
		}
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC || event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			if currentProject != "" {
				currentProject = ""
				updateFunc()
				return nil
			}
		}
		return event
	})

	updateFunc = func() {
		logs := getLogsFromAllFiles()
		table.Clear()

		if currentProject == "" {
			// SHOW PROJECTS LIST
			headers := []string{"Project Name (Press Enter to Drill-down)", "Logs Count", "Total Cost ($)", "Last Active"}
			for i, h := range headers {
				table.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false).SetExpansion(1))
			}

			type projStats struct {
				count int
				cost  float64
				last  string
			}
			stats := make(map[string]*projStats)

			for _, l := range logs {
				proj := filepath.Base(l.Workspace)
				if proj == "." || proj == "/" {
					proj = "Unknown"
				}
				if stats[proj] == nil {
					stats[proj] = &projStats{}
				}
				stats[proj].count++
				stats[proj].cost += l.Cost
				if l.Timestamp > stats[proj].last {
					stats[proj].last = l.Timestamp
				}
			}

			type kv struct {
				Key   string
				Value *projStats
			}
			var ss []kv
			for k, v := range stats {
				ss = append(ss, kv{k, v})
			}
			sort.Slice(ss, func(i, j int) bool {
				return ss[i].Value.count > ss[j].Value.count
			})

			for row, kv := range ss {
				table.SetCell(row+1, 0, tview.NewTableCell(kv.Key).SetTextColor(tcell.ColorGreen))
				table.SetCell(row+1, 1, tview.NewTableCell(fmt.Sprintf("%d", kv.Value.count)).SetTextColor(tcell.ColorWhite))
				table.SetCell(row+1, 2, tview.NewTableCell(fmt.Sprintf("$%.4f", kv.Value.cost)).SetTextColor(tcell.ColorRed))
				table.SetCell(row+1, 3, tview.NewTableCell(kv.Value.last).SetTextColor(tcell.ColorGray))
			}
		} else {
			// SHOW PER-FILE LIST FOR PROJECT
			headers := []string{fmt.Sprintf("Files in [%s] (Press ESC to go back)", currentProject), "Logs Count", "Total Tokens", "Total Cost ($)"}
			for i, h := range headers {
				table.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false).SetExpansion(1))
			}

			type fileStats struct {
				count  int
				tokens int
				cost   float64
			}
			stats := make(map[string]*fileStats)

			for _, l := range logs {
				proj := filepath.Base(l.Workspace)
				if proj == "." || proj == "/" {
					proj = "Unknown"
				}
				if proj == currentProject {
					for _, f := range l.Files {
						if stats[f] == nil {
							stats[f] = &fileStats{}
						}
						stats[f].count++
						stats[f].tokens += l.TokensIn + l.TokensOut
						// Rough estimate of file cost (divide log cost equally by files changed)
						stats[f].cost += l.Cost / float64(len(l.Files))
					}
				}
			}

			type kv struct {
				Key   string
				Value *fileStats
			}
			var ss []kv
			for k, v := range stats {
				ss = append(ss, kv{k, v})
			}
			sort.Slice(ss, func(i, j int) bool {
				return ss[i].Value.cost > ss[j].Value.cost
			})

			for row, kv := range ss {
				table.SetCell(row+1, 0, tview.NewTableCell(kv.Key).SetTextColor(tcell.ColorGreen))
				table.SetCell(row+1, 1, tview.NewTableCell(fmt.Sprintf("%d", kv.Value.count)).SetTextColor(tcell.ColorWhite))
				table.SetCell(row+1, 2, tview.NewTableCell(fmt.Sprintf("%d", kv.Value.tokens)).SetTextColor(tcell.ColorBlue))
				table.SetCell(row+1, 3, tview.NewTableCell(fmt.Sprintf("$%.4f", kv.Value.cost)).SetTextColor(tcell.ColorRed))
			}
		}
	}

	updateFunc()
	return tview.NewFrame(table).SetBorders(0, 0, 0, 0, 0, 0), updateFunc
}

func createTagsTab() (tview.Primitive, func()) {
	chartView := tview.NewTextView().SetDynamicColors(true).SetWrap(false)
	chartView.SetBorder(true).SetTitle(" Tag Analytics ")

	updateFunc := func() {
		logs := getLogsFromAllFiles()
		tagCounts := make(map[string]int)
		for _, l := range logs {
			for _, t := range l.Tags {
				tagCounts[t]++
			}
		}

		chartView.Clear()
		if len(tagCounts) == 0 {
			fmt.Fprintf(chartView, "\n  [gray]No tags used yet.[white]")
			return
		}

		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range tagCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

		maxVal := 0
		for _, item := range sorted {
			if item.v > maxVal {
				maxVal = item.v
			}
		}

		fmt.Fprintf(chartView, "\n [yellow]Top Tags Usage Bar Chart[white]\n\n")
		for i, item := range sorted {
			if i > 20 {
				break // Only show top 20
			}
			barLen := int(float64(item.v) / float64(maxVal) * 40.0)
			bar := strings.Repeat("█", maxVal)
			if barLen < len(bar) { bar = strings.Repeat("█", barLen) }
			fmt.Fprintf(chartView, " %-15s | [cyan]%s[white] (%d)\n", item.k, bar, item.v)
		}
	}
	updateFunc()
	return chartView, updateFunc
}
