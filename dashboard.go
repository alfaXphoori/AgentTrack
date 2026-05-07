package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func runDashboard() {
	loadConfig()

	// Set global tview styles for a dark theme
	tview.Styles.PrimitiveBackgroundColor = tcell.ColorBlack
	tview.Styles.ContrastBackgroundColor = tcell.ColorBlack
	tview.Styles.MoreContrastBackgroundColor = tcell.ColorBlack

	app := tview.NewApplication()

	// Header
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprintf("[yellow::b]TrackCLI Dashboard[white] | [green]%s", time.Now().Format("2006-01-02 15:04")))

	// Pages for the tabs
	infoPages := tview.NewPages()

	tabOverview := createOverviewTab()
	tabLogs := createLogsTab(app)
	tabStats := createStatsTab()
	tabTags := createTagsTab()
	tabWatch := createWatchTab(app)

	infoPages.AddPage("Overview", tabOverview, true, true)
	infoPages.AddPage("Logs", tabLogs, true, false)
	infoPages.AddPage("Stats", tabStats, true, false)
	infoPages.AddPage("Tags", tabTags, true, false)
	infoPages.AddPage("Live", tabWatch, true, false)

	// Tab bar
	tabBar := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)
	
	// Create the tab text
	tabs := []string{"Overview", "Logs", "Stats", "Tags", "Live"}
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
	
	updateTabBar("Overview")
	tabBar.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			infoPages.SwitchToPage(added[0])
			updateTabBar(added[0])
		}
	})

	// Footer
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]1-5[white] Tabs | [yellow]↑/↓[white] Navigate | [yellow]Enter[white] View | [yellow]q/Ctrl+C[white] Quit")

	// Global key handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				app.Stop()
				return nil
			case '1':
				tabBar.Highlight("Overview")
			case '2':
				tabBar.Highlight("Logs")
			case '3':
				tabBar.Highlight("Stats")
			case '4':
				tabBar.Highlight("Tags")
			case '5':
				tabBar.Highlight("Live")
			}
		}
		return event
	})

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(tabBar, 1, 0, false).
		AddItem(infoPages, 0, 1, true).
		AddItem(footer, 1, 0, false)

	if err := app.SetRoot(layout, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func createOverviewTab() tview.Primitive {
	logs := getLogsFromAllFiles()
	today := time.Now().Format("2006-01-02")
	
	var todayLogs []LogEntry
	tIn, tOut := 0, 0
	cost := 0.0

	models := make(map[string]int)

	for _, l := range logs {
		if strings.HasPrefix(l.Timestamp, today) {
			todayLogs = append(todayLogs, l)
			tIn += l.TokensIn
			tOut += l.TokensOut
			if c, ok := calculateLogCost(l); ok {
				cost += c
			}
		}
		m := logModel(l)
		if m != "" {
			models[m]++
		}
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Today snapshot
	snapshotText := fmt.Sprintf(`[cyan::b]Today's Snapshot[white::-]
Logs Recorded: [green]%d[white]
Total Tokens:  [yellow]%d[white] (In: %d, Out: %d)
Estimated Cost: [green]%s %.4f[white]`, len(todayLogs), tIn+tOut, tIn, tOut, config.Pricing.Currency, cost)
	
	snapshotView := tview.NewTextView().SetDynamicColors(true).SetText(snapshotText)
	snapshotView.SetBorder(true).SetTitle(" Today ")

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

	// Recent Activity
	recentText := ""
	limit := 10
	if len(logs) < limit {
		limit = len(logs)
	}
	
	for i := len(logs) - 1; i >= len(logs)-limit; i-- {
		l := logs[i]
		msg := l.Message
		if l.Category == "AutoLog" && l.Question != "" {
			msg = l.Question
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
		}
		recentText += fmt.Sprintf("[cyan]%s[white] | [yellow]%-15s[white] | %s\n", l.Timestamp[11:16], logModel(l), msg)
	}
	
	recentView := tview.NewTextView().SetDynamicColors(true).SetText(recentText)
	recentView.SetBorder(true).SetTitle(" Recent Activity ")

	topRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(snapshotView, 0, 1, false).
		AddItem(modelsView, 0, 1, false)

	flex.AddItem(topRow, 7, 1, false).
		AddItem(recentView, 0, 2, false)

	return flex
}

func createLogsTab(app *tview.Application) tview.Primitive {
	logs := getLogsFromAllFiles()
	
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)

	headers := []string{"Time", "Model", "Category", "Tokens", "Time", "Content"}
	for i, h := range headers {
		table.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false).SetExpansion(1))
	}

	row := 1
	for i := len(logs) - 1; i >= 0; i-- {
		l := logs[i]
		
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

		table.SetCell(row, 0, tview.NewTableCell(l.Timestamp[5:16]).SetTextColor(tcell.ColorTeal))
		table.SetCell(row, 1, tview.NewTableCell(logModel(l)).SetTextColor(tcell.ColorBlue))
		table.SetCell(row, 2, tview.NewTableCell(logCategory(l)).SetTextColor(tcell.ColorPurple))
		table.SetCell(row, 3, tview.NewTableCell(tokens).SetTextColor(tcell.ColorYellow))
		table.SetCell(row, 4, tview.NewTableCell(duration).SetTextColor(tcell.ColorGreen))
		table.SetCell(row, 5, tview.NewTableCell(msg).SetTextColor(tcell.ColorWhite))
		row++
	}

	// Setup layout with a side panel for details
	detailView := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetWordWrap(true)
	detailView.SetBorder(true).SetTitle(" Detail ")

	table.SetSelectionChangedFunc(func(row, column int) {
		if row == 0 {
			detailView.SetText("")
			return
		}
		logIdx := len(logs) - row // Reverse mapping
		if logIdx >= 0 && logIdx < len(logs) {
			l := logs[logIdx]
			
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
	})

	if len(logs) > 0 {
		table.Select(1, 0)
	}

	flex := tview.NewFlex().
		AddItem(table, 0, 2, true).
		AddItem(detailView, 0, 1, false)

	return flex
}

func createStatsTab() tview.Primitive {
	logs := getLogsFromAllFiles()
	rows := collectModelStats(logs)

	table := tview.NewTable().SetBorders(true).SetSelectable(true, false)
	headers := []string{"Model", "Logs", "Tokens In", "Tokens Out", "Total Tokens", "Estimated Cost"}
	for i, h := range headers {
		table.SetCell(0, i, tview.NewTableCell(h).SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	}

	for i, r := range rows {
		cost := "n/a"
		if r.HasCost {
			cost = fmt.Sprintf("%s %.4f", config.Pricing.Currency, r.Cost)
		}
		
		table.SetCell(i+1, 0, tview.NewTableCell(r.Model).SetTextColor(tcell.ColorBlue))
		table.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%d", r.Logs)).SetAlign(tview.AlignRight))
		table.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%d", r.TokensIn)).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorYellow))
		table.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%d", r.TokensOut)).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorYellow))
		table.SetCell(i+1, 4, tview.NewTableCell(fmt.Sprintf("%d", r.TokensIn+r.TokensOut)).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorRed))
		table.SetCell(i+1, 5, tview.NewTableCell(cost).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText(" Overall Usage by Model ").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(table, 0, 1, true)

	return flex
}

func createTagsTab() tview.Primitive {
	logs := getLogsFromAllFiles()
	
	tagCounts := map[string]int{}
	catCounts := map[string]int{}

	for _, l := range logs {
		for _, tag := range l.Tags {
			tagCounts[tag]++
		}
		catCounts[logCategory(l)]++
	}

	// Tags table
	type kv struct {
		k string
		v int
	}
	var sortedTags []kv
	for k, v := range tagCounts {
		sortedTags = append(sortedTags, kv{k, v})
	}
	sort.Slice(sortedTags, func(i, j int) bool { return sortedTags[i].v > sortedTags[j].v })

	tagsList := tview.NewList().ShowSecondaryText(false)
	tagsList.SetBorder(true).SetTitle(" Tags Cloud ")
	for _, t := range sortedTags {
		tagsList.AddItem(fmt.Sprintf("%s (%d)", t.k, t.v), "", 0, nil)
	}

	// Category table
	var sortedCats []kv
	for k, v := range catCounts {
		sortedCats = append(sortedCats, kv{k, v})
	}
	sort.Slice(sortedCats, func(i, j int) bool { return sortedCats[i].v > sortedCats[j].v })

	catsList := tview.NewList().ShowSecondaryText(false)
	catsList.SetBorder(true).SetTitle(" Categories ")
	for _, c := range sortedCats {
		catsList.AddItem(fmt.Sprintf("%s (%d)", c.k, c.v), "", 0, nil)
	}

	flex := tview.NewFlex().
		AddItem(catsList, 0, 1, true).
		AddItem(tagsList, 0, 2, false)

	return flex
}

func createWatchTab(app *tview.Application) tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	
	textView.SetBorder(true).SetTitle(" Live Watch ")
	
	fmt.Fprintf(textView, "[cyan]👀 Watching for new TrackCLI logs in real-time...[white]\n\n")

	go func() {
		lastCount := len(getLogsFromAllFiles())
		for {
			time.Sleep(1 * time.Second)
			logs := getLogsFromAllFiles()
			currentCount := len(logs)

			if currentCount > lastCount {
				for i := lastCount; i < currentCount; i++ {
					log := logs[i]
					category := logCategory(log)
					
					durStr := ""
					if log.Duration > 0 {
						durStr = fmt.Sprintf(" | Time: %.2fs", log.Duration)
					}
					
					metadata := fmt.Sprintf("[yellow][Model: %s | Tokens: In=%d, Out=%d%s][white]", logModel(log), log.TokensIn, log.TokensOut, durStr)

					app.QueueUpdateDraw(func() {
						if category == "AutoLog" && log.Question != "" {
							fmt.Fprintf(textView, "✨ [green]New Log:[white] [[cyan]%s[white]] | %s\n", log.Timestamp, metadata)
							fmt.Fprintf(textView, "  👤 [white::b]Q: %s[-::-]\n", log.Question)
							fmt.Fprintf(textView, "  🤖 [white::b]A: %s[-::-]\n", log.Answer)
						} else {
							fmt.Fprintf(textView, "✨ [green]New Log:[white] [[cyan]%s[white]] ([purple]%s[white]) | 📝 %s\n", log.Timestamp, category, log.Message)
						}
						fmt.Fprintf(textView, "[gray]%s[white]\n", strings.Repeat("-", 80))
						textView.ScrollToEnd()
					})
				}
				lastCount = currentCount
			} else if currentCount < lastCount {
				lastCount = currentCount
			}
		}
	}()

	return textView
}

