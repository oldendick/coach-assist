package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/oldendick/coach-assist/internal/config"
	"github.com/oldendick/coach-assist/internal/domain"
	"github.com/oldendick/coach-assist/internal/drive"
	"github.com/oldendick/coach-assist/internal/ingest"
	"github.com/oldendick/coach-assist/internal/state"
)

// isRowReference checks if a cell exists at row/col and has the expected reference string.
// This prevents nil panics when navigating sparsely populated or empty tables.
func isRowReference(table *tview.Table, row, col int, ref string) bool {
	if row < 0 || row >= table.GetRowCount() {
		return false
	}
	cell := table.GetCell(row, col)
	if cell == nil {
		return false
	}
	r := cell.GetReference()
	if r == nil {
		return false
	}
	s, ok := r.(string)
	return ok && s == ref
}

func RunTUI(cfg *config.AppConfig, driveSvc drive.WorkspaceService, version string) {
	app := tview.NewApplication()
	pages := tview.NewPages()

	appState := state.LoadState("state.json")
	var assignments []domain.FlightPlan
	var allScheduleRows []domain.ScheduleRow
	folderStatusChecked := false

	// loadAssignments attempts to parse both Excel files and build real assignments.
	// Returns silently if either file is missing (they just haven't been synced yet).
	loadAssignments := func() {
		schedPath := filepath.Join("artifacts", "latest-schedule.xlsx")
		rosterPath := filepath.Join("artifacts", "latest-roster.xlsx")

		schedRows, err := ingest.ParseSchedule(schedPath)
		if err != nil {
			return
		}

		groupMap, _ := ingest.ParseGroupAssignments(rosterPath)
		emailMap, _ := ingest.ParseStudentEmails(rosterPath)

		assignments = domain.BuildAssignments(
			cfg.Coaches[cfg.ActiveCoach].Name,
			schedRows,
			groupMap,
			emailMap,
		)
		allScheduleRows = schedRows

		// Merge persistent state
		foundCached := false
		for i := range assignments {
			for _, cached := range appState.Assignments {
				if assignments[i].SubjectName == cached.SubjectName && assignments[i].ArrivalDay == cached.ArrivalDay {
					assignments[i].IsDiscoveryDrafted = cached.IsDiscoveryDrafted
					assignments[i].IsFinalPlanDrafted = cached.IsFinalPlanDrafted
					assignments[i].IsFollowUpSent = cached.IsFollowUpSent
					// Also carry over Drive state if it was already resolved
					if cached.DriveFileID != "" {
						assignments[i].DriveFileID = cached.DriveFileID
						assignments[i].IsWorkspaceCreated = cached.IsWorkspaceCreated
						assignments[i].HasTrainingPlan = cached.HasTrainingPlan
						foundCached = true
					}
				}
			}
		}
		if foundCached {
			folderStatusChecked = true
		}
	}

	// Try loading on boot if files exist
	loadAssignments()

	list := tview.NewList()
	list.SetBorder(true).SetTitle(" Main Menu ").SetTitleAlign(tview.AlignLeft)

	mainContainer := tview.NewFlex().SetDirection(tview.FlexRow)
	mainContainer.SetBorder(true).
		SetTitle(fmt.Sprintf(" COACH ASSIST %s - Active Profile: %s ", version, cfg.ActiveCoach)).
		SetTitleAlign(tview.AlignLeft)

	table := tview.NewTable().SetBorders(true).SetSelectable(true, false).SetFixed(1, 0)
	table.SetBorder(true).SetTitle(" Pending Assignments (r: refresh folders, q/ESC: back, Enter: view) ").SetTitleAlign(tview.AlignLeft)
	table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite))

	masterScheduleTable := tview.NewTable().SetBorders(true).SetSelectable(true, true).SetFixed(1, 0)
	masterScheduleTable.SetBorder(true).SetTitle(" Master Schedule (ESC/q: back) ").SetTitleAlign(tview.AlignLeft)
	masterScheduleTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite))

	coachScheduleTable := tview.NewTable().SetBorders(true).SetSelectable(true, true).SetFixed(1, 0)
	coachScheduleTable.SetBorder(true)
	coachScheduleTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite))

	refreshTable := func() {
		table.Clear()
		table.SetCell(0, 0, tview.NewTableCell("Student / Group").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetExpansion(1))
		table.SetCell(0, 1, tview.NewTableCell("Dive Coach").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 2, tview.NewTableCell("Day").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 3, tview.NewTableCell("Time").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 4, tview.NewTableCell("Folder").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 5, tview.NewTableCell("Plan").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 6, tview.NewTableCell("Outreach").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 7, tview.NewTableCell("Plan Sent").SetSelectable(false).SetTextColor(tcell.ColorYellow))
		table.SetCell(0, 8, tview.NewTableCell("Follow-up").SetSelectable(false).SetTextColor(tcell.ColorYellow))

		// Bucket assignments into categories
		var solos, groups, reserved []domain.FlightPlan
		for _, plan := range assignments {
			if plan.IsReserved {
				reserved = append(reserved, plan)
			} else if plan.IsGroup {
				groups = append(groups, plan)
			} else {
				solos = append(solos, plan)
			}
		}

		row := 1
		addSection := func(label string, color tcell.Color, plans []domain.FlightPlan) {
			if len(plans) == 0 {
				return
			}
			// Section header
			table.SetCell(row, 0, tview.NewTableCell(label).SetSelectable(false).SetTextColor(color).SetExpansion(1))
			table.SetCell(row, 1, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 4, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 5, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 6, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 7, tview.NewTableCell("").SetSelectable(false))
			table.SetCell(row, 8, tview.NewTableCell("").SetSelectable(false))
			row++

			for _, plan := range plans {
				folderStatus := "❓"
				planStatus := "❓"
				if plan.IsReserved {
					folderStatus = "⏳"
					planStatus = "⏳"
				} else if folderStatusChecked {
					if plan.IsWorkspaceCreated {
						folderStatus = "✅"
					} else {
						folderStatus = "❌"
					}
					if plan.HasTrainingPlan {
						planStatus = "✅"
					} else {
						planStatus = "❌"
					}
				}
				nameCell := tview.NewTableCell("  " + plan.SubjectName).SetReference("assignment")
				if plan.IsReserved {
					nameCell.SetTextColor(tcell.ColorDarkGray)
				}
				table.SetCell(row, 0, nameCell)
				table.SetCell(row, 1, tview.NewTableCell(plan.MakingDivesCoach))
				table.SetCell(row, 2, tview.NewTableCell(plan.ArrivalDay))
				table.SetCell(row, 3, tview.NewTableCell(plan.ArrivalTime))
				table.SetCell(row, 4, tview.NewTableCell(folderStatus))
				table.SetCell(row, 5, tview.NewTableCell(planStatus))

				outreachIcon := "❌"
				if plan.IsDiscoveryDrafted {
					outreachIcon = "✅"
				}
				planSentIcon := "❌"
				if plan.IsFinalPlanDrafted {
					planSentIcon = "✅"
				}
				followUpIcon := "❌"
				if plan.IsFollowUpSent {
					followUpIcon = "✅"
				}
				table.SetCell(row, 6, tview.NewTableCell(outreachIcon).SetAlign(tview.AlignCenter))
				table.SetCell(row, 7, tview.NewTableCell(planSentIcon).SetAlign(tview.AlignCenter))
				table.SetCell(row, 8, tview.NewTableCell(followUpIcon).SetAlign(tview.AlignCenter))
				row++
			}
		}

		addSection("── Group Teams ──", tcell.ColorGreen, groups)
		addSection("── 1-on-1 Students ──", tcell.ColorAqua, solos)
		addSection("── Reserved Slots ──", tcell.ColorDarkGray, reserved)

		if row == 1 {
			// No plans added - add a selectable placeholder to ensure navigation doesn't hang
			table.SetCell(1, 0, tview.NewTableCell("  No pending assignments found.").SetSelectable(true).SetExpansion(1))
		}

		// Auto-select first data row (below the first header)
		if table.GetRowCount() > 1 {
			table.Select(1, 0)
		}
	}

	// === ASSIGNMENT DETAIL VIEW ===
	detailPage := tview.NewFlex().SetDirection(tview.FlexRow)
	detailPage.SetBorder(true).SetTitle(" Assignment Detail View ").SetTitleAlign(tview.AlignLeft)

	// Internal pages to swap between information text and the spreadsheet table
	detailBody := tview.NewPages()

	detailMenu := tview.NewList().ShowSecondaryText(false)
	detailMenu.SetBorder(true).SetTitle(" Assignment Options ")
	detailBody.AddPage("Menu", detailMenu, true, true)

	detailText := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter)
	detailText.SetText("\n\n(Assignment Detail View - Work in Progress)\n\nPress ESC or 'q' to return to the dashboard.")

	detailBody.AddPage("Text", detailText, true, false)

	// We'll add the "Table" page dynamically or just keep it ready
	detailTable := tview.NewTable().SetBorders(true).SetSelectable(true, true).SetFixed(1, 0)
	detailTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkCyan).Foreground(tcell.ColorWhite))
	detailBody.AddPage("Table", detailTable, true, false)

	// Drafting Page
	draftingPage := tview.NewFlex().SetDirection(tview.FlexColumn)
	templateList := tview.NewList().ShowSecondaryText(false)
	templateList.SetBorder(true).SetTitle(" Templates ")
	previewText := tview.NewTextView().SetDynamicColors(true).SetWrap(true).SetRegions(true)
	previewText.SetBorder(true).SetTitle(" Draft Preview ")

	createButton := tview.NewButton("Create Gmail Draft")
	createButton.SetBorder(true)

	rightPane := tview.NewFlex().SetDirection(tview.FlexRow)
	rightPane.AddItem(previewText, 0, 1, false)
	rightPane.AddItem(createButton, 3, 0, false)

	draftingPage.AddItem(templateList, 30, 1, true)
	draftingPage.AddItem(rightPane, 0, 3, false)
	detailBody.AddPage("Drafting", draftingPage, true, false)

	templateList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			detailBody.SwitchToPage("Menu")
			app.SetFocus(detailMenu)
			return nil
		}
		if event.Key() == tcell.KeyRight || event.Key() == tcell.KeyTab {
			app.SetFocus(previewText)
			return nil
		}
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(createButton)
			return nil
		}
		return event
	})

	previewText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			detailBody.SwitchToPage("Menu")
			app.SetFocus(detailMenu)
			return nil
		}
		if event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyBacktab {
			app.SetFocus(templateList)
			return nil
		}
		if event.Key() == tcell.KeyDown || event.Key() == tcell.KeyTab {
			app.SetFocus(createButton)
			return nil
		}
		return event
	})

	createButton.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			detailBody.SwitchToPage("Menu")
			app.SetFocus(detailMenu)
			return nil
		}
		if event.Key() == tcell.KeyUp || event.Key() == tcell.KeyBacktab {
			app.SetFocus(previewText)
			return nil
		}
		if event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyTab {
			app.SetFocus(templateList)
			return nil
		}
		return event
	})

	detailPage.AddItem(detailBody, 0, 1, true)

	table.SetSelectedFunc(func(row, column int) {
		if row < 1 || row >= table.GetRowCount() {
			return
		}

		// 1. Map the UI row back to our assignments data
		displayName := strings.TrimSpace(table.GetCell(row, 0).Text)
		var targetPlan *domain.FlightPlan
		for i := range assignments {
			if assignments[i].SubjectName == displayName {
				targetPlan = &assignments[i]
				break
			}
		}

		if targetPlan == nil {
			pages.SwitchToPage("AssignmentDetail")
			detailBody.SwitchToPage("Text")
			detailText.SetText(fmt.Sprintf("\n\n[red]Internal Error: Could not find assignment '%s' in memory.[-]\n\nPress ESC or 'q' to return.", displayName))
			app.SetFocus(detailPage)
			return
		}

		// 2. Prepare the Assignment Sub-Menu
		detailMenu.Clear()
		detailPage.SetTitle(fmt.Sprintf(" Assignment Detail: %s ", targetPlan.SubjectName))

		// Option 1: Create Drive Workspace
		createLabel := "Create Drive Workspace"
		if targetPlan.HasTrainingPlan {
			createLabel = "[green][✅] Create Drive Workspace[-] (Already Exists)"
		}

		detailMenu.AddItem(createLabel, "Initialize folder and template on Google Drive", 'c', func() {
			detailBody.SwitchToPage("Text")
			detailText.SetText("\n\n[yellow][-] Step 1: Evaluating Drive Hierarchy...[-]\n\n")

			go func() {
				// 1. Determine Parents & Templates
				parentID := cfg.Drive.WorkshopParentFolderID
				templateID := cfg.Drive.Templates.IndividualSkillsWorksheetID
				if targetPlan.IsGroup {
					parentID = cfg.Drive.TeamsFolderID
					templateID = cfg.Drive.Templates.TeamTrainingPlanID
				}

				// 2. Search for existing folder
				app.QueueUpdateDraw(func() {
					detailText.SetText(fmt.Sprintf("\n\n[yellow][-] Step 2: Searching for folder '%s'...[-]\n\n", targetPlan.SubjectName))
				})

				query := fmt.Sprintf("name = '%s' and '%s' in parents and mimeType = 'application/vnd.google-apps.folder' and trashed = false", targetPlan.SubjectName, parentID)
				matches, err := driveSvc.SearchFiles(query)

				var folderID string
				if err == nil && len(matches) > 0 {
					folderID = matches[0].ID
					app.QueueUpdateDraw(func() {
						detailText.SetText(fmt.Sprintf("\n\n[yellow][-] Step 2: Existing folder found (ID: %s). Skipping creation.[-]\n\n", folderID))
					})
				} else {
					app.QueueUpdateDraw(func() {
						detailText.SetText("\n\n[yellow][-] Step 2: Folder not found. Creating new workspace...[-]\n\n")
					})
					folderID, err = driveSvc.CreateFolder(parentID, targetPlan.SubjectName)
				}

				if err != nil || folderID == "" {
					app.QueueUpdateDraw(func() {
						detailText.SetText(fmt.Sprintf("\n\n[red]Failed to resolve workspace folder: %v[-]\n\nPress ESC or 'q' to return.", err))
					})
					return
				}

				// 3. Copy Template
				app.QueueUpdateDraw(func() {
					detailText.SetText("\n\n[yellow][-] Step 3: Cloning Template into workspace...[-]\n\n")
				})
				newName := fmt.Sprintf("%s - Training Plan", targetPlan.SubjectName)
				newFileID, err := driveSvc.CopyFile(templateID, folderID, newName)
				if err != nil {
					app.QueueUpdateDraw(func() {
						detailText.SetText(fmt.Sprintf("\n\n[red]Failed to clone template: %v[-]\n\nPress ESC or 'q' to return.", err))
					})
					return
				}

				// 4. Set Permissions
				app.QueueUpdateDraw(func() {
					detailText.SetText("\n\n[yellow][-] Step 4: Applying 'Anyone with Link' Reader permissions...[-]\n\n")
				})
				err = driveSvc.CreatePermission(newFileID, "reader", "anyone")
				if err != nil {
					app.QueueUpdateDraw(func() {
						detailText.SetText(fmt.Sprintf("\n\n[red]Failed to set permissions: %v[-]\n\nPress ESC or 'q' to return.", err))
					})
					return
				}

				// 5. Personalize Template
				app.QueueUpdateDraw(func() {
					detailText.SetText("\n\n[yellow][-] Step 5: Dynamically mapping worksheet grid...[-]\n\n")
				})

				meta, err := driveSvc.GetSpreadsheetMetadata(newFileID)
				if err == nil {
					var sheetName string
					isTeamSheet := false
					for _, s := range meta.Sheets {
						title := strings.ToLower(s.Properties.Title)
						if !targetPlan.IsGroup && (strings.Contains(title, "plan") || strings.Contains(title, "skills")) {
							sheetName = s.Properties.Title
							break
						}
						if targetPlan.IsGroup && strings.Contains(title, "everyone") {
							sheetName = s.Properties.Title
							isTeamSheet = true
							break
						}
					}

					if sheetName == "" && len(meta.Sheets) > 0 {
						sheetName = meta.Sheets[0].Properties.Title
					}

					if sheetName != "" {
						grid, err := driveSvc.GetSheetValues(newFileID, fmt.Sprintf("%s!A1:Z50", sheetName))
						if err == nil {
							var merges []drive.SheetMerge
							for _, s := range meta.Sheets {
								if s.Properties.Title == sheetName {
									merges = s.Merges
									break
								}
							}

							displayDate := targetPlan.ArrivalDay
							if displayDate == "" {
								displayDate = time.Now().Format("2006-01-02")
							}

							var updates []drive.SheetUpdate
							if !isTeamSheet {
								for rIdx, row := range grid {
									for cIdx, cell := range row {
										cellStr := strings.ToLower(strings.TrimSpace(fmt.Sprint(cell)))
										var targetColIdx int
										if strings.Contains(cellStr, "name:") {
											targetColIdx = getTargetCol(merges, rIdx, cIdx)
											updates = append(updates, drive.SheetUpdate{Range: fmt.Sprintf("%s!%s%d", sheetName, colNumToLetter(targetColIdx+1), rIdx+1), Values: [][]interface{}{{targetPlan.SubjectName}}})
										} else if strings.Contains(cellStr, "date:") {
											targetColIdx = getTargetCol(merges, rIdx, cIdx)
											updates = append(updates, drive.SheetUpdate{Range: fmt.Sprintf("%s!%s%d", sheetName, colNumToLetter(targetColIdx+1), rIdx+1), Values: [][]interface{}{{displayDate}}})
										} else if strings.Contains(cellStr, "coach:") {
											targetColIdx = getTargetCol(merges, rIdx, cIdx)
											updates = append(updates, drive.SheetUpdate{Range: fmt.Sprintf("%s!%s%d", sheetName, colNumToLetter(targetColIdx+1), rIdx+1), Values: [][]interface{}{{cfg.ActiveCoach}}})
										}
									}
								}
							} else {
								updates = append(updates, drive.SheetUpdate{Range: fmt.Sprintf("%s!A3", sheetName), Values: [][]interface{}{{displayDate}}})
							}

							if len(updates) > 0 {
								driveSvc.UpdateSheetValues(newFileID, updates)
							}
						}
					}
				}

				app.QueueUpdateDraw(func() {
					// Update local memory state so "View" and "Table" work immediately
					targetPlan.HasTrainingPlan = true
					targetPlan.IsWorkspaceCreated = true
					targetPlan.DriveFileID = newFileID
					folderStatusChecked = true

					// Update the menu label immediately
					detailMenu.SetItemText(0, "[green][✅] Create Drive Workspace[-] (Already Exists)", "Initialize folder and template on Google Drive")

					// Update the dashboard table in the background
					refreshTable()

					successMsg := fmt.Sprintf("\n\n[green]Workspace Successfully Created![-]\n\n[white]Folder Created:[-] %s\n[white]Plan Created:[-] %s\n\nLink: https://docs.google.com/spreadsheets/d/%s/edit\n\nPress ESC or 'q' to return to options.", targetPlan.SubjectName, newName, newFileID)
					detailText.SetText(successMsg)
				})
			}()
		})

		detailMenu.AddItem("", "", 0, nil) // Spacer at index 1

		// Option: View Filtered Schedule
		detailMenu.AddItem("View Filtered Schedule", "Show all flight times for this student/group", 'v', func() {
			detailBody.SwitchToPage("Table")
			detailTable.Clear()
			detailTable.SetCell(0, 0, tview.NewTableCell("Date").SetTextColor(tcell.ColorYellow))
			detailTable.SetCell(0, 1, tview.NewTableCell("Time").SetTextColor(tcell.ColorYellow))
			detailTable.SetCell(0, 2, tview.NewTableCell("Coach 1").SetTextColor(tcell.ColorYellow))
			detailTable.SetCell(0, 3, tview.NewTableCell("Group 1").SetTextColor(tcell.ColorYellow))
			detailTable.SetCell(0, 4, tview.NewTableCell("Coach 2").SetTextColor(tcell.ColorYellow))
			detailTable.SetCell(0, 5, tview.NewTableCell("Group 2").SetTextColor(tcell.ColorYellow))

			rowIdx := 1
			searchName := strings.TrimSpace(targetPlan.SubjectName)
			for _, sched := range allScheduleRows {
				if strings.Contains(sched.Group1, searchName) || strings.Contains(sched.Group2, searchName) {
					detailTable.SetCell(rowIdx, 0, tview.NewTableCell(sched.Date).SetReference("data"))
					detailTable.SetCell(rowIdx, 1, tview.NewTableCell(sched.FlyingAt))
					detailTable.SetCell(rowIdx, 2, tview.NewTableCell(sched.Coach1))
					detailTable.SetCell(rowIdx, 3, tview.NewTableCell(sched.Group1))
					detailTable.SetCell(rowIdx, 4, tview.NewTableCell(sched.Coach2))
					detailTable.SetCell(rowIdx, 5, tview.NewTableCell(sched.Group2))
					rowIdx++
				}
			}
			if rowIdx == 1 {
				detailTable.SetCell(1, 0, tview.NewTableCell("No schedule entries found.").SetExpansion(1))
			}
			detailTable.Select(0, 0)
			detailTable.ScrollToBeginning()
			app.SetFocus(detailTable)
		})

		detailMenu.AddItem("", "", 0, nil) // Spacer at index 1

		// Option 2: View Training Plan
		detailMenu.AddItem("View Training Plan", "Download and render spreadsheet", 'p', func() {
			if !targetPlan.HasTrainingPlan || targetPlan.DriveFileID == "" {
				detailBody.SwitchToPage("Text")
				detailText.SetText("\n\n(No training plan found for this assignment yet)\n\nPress ESC or 'q' to return to options.")
				return
			}

			detailBody.SwitchToPage("Text")
			detailText.SetText("\n\n[yellow][-] Step 1: Initiating Google Drive handshake...[-]\n\n")

			go func() {
				planDir := filepath.Join("artifacts", "plans")
				os.MkdirAll(planDir, 0755)
				destPath := filepath.Join(planDir, domain.SanitizeFileName(targetPlan.SubjectName)+".xlsx")

				app.QueueUpdateDraw(func() {
					detailText.SetText(fmt.Sprintf("\n\n[yellow][-] Step 2: Exporting Spreadsheet: %s...[-]\n\n", targetPlan.DriveFileID))
				})

				err := driveSvc.ExportFile(targetPlan.DriveFileID, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", destPath)
				if err != nil {
					app.QueueUpdateDraw(func() {
						detailText.SetText("\n\n[yellow][-] Step 2b: Export failed, attempting raw binary download...[-]\n\n")
					})
					err = driveSvc.DownloadFile(targetPlan.DriveFileID, destPath)
				}

				app.QueueUpdateDraw(func() {
					if err != nil {
						detailText.SetText(fmt.Sprintf("\n\n[red]Failed to load training plan: %v[-]\n\nPress ESC or 'q' to return to options.", err))
						return
					}
					detailText.SetText("\n\n[yellow][-] Step 3: Parsing Excel artifacts (locally)...[-]\n\n")
				})

				rows, err := ingest.ReadRawExcel(destPath, "")

				app.QueueUpdateDraw(func() {
					if err != nil {
						detailText.SetText(fmt.Sprintf("\n\n[red]Failed to parse training plan: %v[-]\n\nPress ESC or 'q' to return into options.", err))
						return
					}
					detailText.SetText(fmt.Sprintf("\n\n[yellow][-] Step 4: Generating TUI Table (%d rows discovered)...[-]\n\n", len(rows)))

					detailTable.Clear()
					limit := 100
					renderRows := len(rows)
					if renderRows > limit {
						renderRows = limit
					}

					trueMaxColWidths := make(map[int]int)
					for r := 0; r < renderRows; r++ {
						for c, cell := range rows[r] {
							if len(cell) > trueMaxColWidths[c] {
								trueMaxColWidths[c] = len(cell)
							}
						}
					}

					for r := 0; r < renderRows; r++ {
						rRows := rows[r]
						for c, cell := range rRows {
							colWidth := trueMaxColWidths[c]
							if colWidth > 30 {
								colWidth = 30
							}
							display := cell
							if len(display) > colWidth {
								display = display[:colWidth-3] + "..."
							}
							padded := fmt.Sprintf(" %-*s ", colWidth, display)
							tableCell := tview.NewTableCell(padded)
							if r == 0 {
								tableCell.SetTextColor(tcell.ColorYellow)
							}
							detailTable.SetCell(r, c, tableCell)
						}
					}

					if len(rows) > limit {
						footerRow := detailTable.GetRowCount()
						detailTable.SetCell(footerRow, 0, tview.NewTableCell(fmt.Sprintf("  ... (Truncated: %d rows total) ...", len(rows))).SetSelectable(false).SetTextColor(tcell.ColorDarkGray))
					}

					detailTable.Select(0, 0)
					detailTable.ScrollToBeginning()
					detailBody.SwitchToPage("Table")
					app.SetFocus(detailTable)
				})
			}()
		})

		detailMenu.AddItem("", "", 0, nil) // Spacer at index 3

		// Option 3: Draft Email
		detailMenu.AddItem("Draft Email", "Preview and customize draft", 'd', func() {
			coach := cfg.Coaches[cfg.ActiveCoach]
			templateList.Clear()

			category := "1on1"
			if targetPlan.IsGroup {
				category = "group"
			}

			templates := make(map[string]config.EmailTemplate)
			for k, v := range coach.EmailTemplates[category] {
				templates[k] = v
			}

			if len(templates) == 0 {
				// Fallback to searching all categories if specific one is empty
				for cat, tmpls := range coach.EmailTemplates {
					for name, tmpl := range tmpls {
						templates[cat+": "+name] = tmpl
					}
				}
			}

			// Sort template names by SortOrder, then by name for stability
			var names []string
			for name := range templates {
				names = append(names, name)
			}
			sort.Slice(names, func(i, j int) bool {
				if templates[names[i]].SortOrder != templates[names[j]].SortOrder {
					return templates[names[i]].SortOrder < templates[names[j]].SortOrder
				}
				return names[i] < names[j]
			})

			var curDraft struct {
				from, to, cc, subj, body string
			}
			var selectedTemplate *config.EmailTemplate

			updateDraftPreview := func(templateName string) {
				tmpl := templates[templateName]
				selectedTemplate = &tmpl
				body := tmpl.Body
				folderLink := "https://docs.google.com/spreadsheets/d/" + targetPlan.DriveFileID + "/edit"
				if targetPlan.DriveFileID == "" {
					folderLink = "[red](No Drive Workspace Created yet)[-]"
				}

				firstName := strings.Split(targetPlan.SubjectName, " ")[0]
				populatedBody := body
				populatedBody = strings.ReplaceAll(populatedBody, "{folder_link}", folderLink)
				populatedBody = strings.ReplaceAll(populatedBody, "{group_name}", targetPlan.SubjectName)
				populatedBody = strings.ReplaceAll(populatedBody, "{name}", targetPlan.SubjectName)
				populatedBody = strings.ReplaceAll(populatedBody, "{firstname}", firstName)
				populatedBody = strings.ReplaceAll(populatedBody, "{initial_meet_time}", targetPlan.ArrivalTime)

				toEmails := strings.Join(targetPlan.SubjectEmails, ", ")
				if toEmails == "" {
					toEmails = "[red](No student emails found)[-]"
				}
				
				ccEmails := ""
				if tmpl.IncludeCC {
					ccEmails = strings.Join(cfg.Workshop.CCEmails, ", ")
				}

				curDraft.from = coach.DraftedFrom
				curDraft.to = toEmails
				curDraft.cc = ccEmails

				subj := tmpl.Subject
				if subj == "" {
					subj = "Training Plan: {name}"
				}
				subj = strings.ReplaceAll(subj, "{name}", targetPlan.SubjectName)
				subj = strings.ReplaceAll(subj, "{groupname}", targetPlan.SubjectName)
				subj = strings.ReplaceAll(subj, "{firstname}", firstName)
				curDraft.subj = subj

				curDraft.body = populatedBody + "\n\n"
				if coach.Signature != "" {
					curDraft.body += coach.Signature
				}

				previewText.Clear()
				fmt.Fprintf(previewText, "[yellow]From:[-] %s\n", curDraft.from)
				fmt.Fprintf(previewText, "[yellow]To:  [-] %s\n", curDraft.to)
				fmt.Fprintf(previewText, "[yellow]CC:  [-] %s\n", curDraft.cc)
				fmt.Fprintf(previewText, "[yellow]Subj:[-] %s\n\n", curDraft.subj)
				fmt.Fprintf(previewText, "--------------------------------------------------\n\n")
				fmt.Fprintf(previewText, "%s", curDraft.body)
				previewText.ScrollToBeginning()
			}

			createButton.SetSelectedFunc(func() {
				if strings.Contains(curDraft.to, "[red]") {
					return
				}

				createButton.SetLabel(" [yellow]Creating Draft... ")
				createButton.SetBackgroundColor(tcell.ColorDarkCyan)

				go func() {
					err := driveSvc.CreateDraft(curDraft.from, curDraft.to, curDraft.cc, curDraft.subj, curDraft.body)
					app.QueueUpdateDraw(func() {
						if err != nil {
							createButton.SetLabel(fmt.Sprintf(" [red]Error: %v ", err))
							createButton.SetBackgroundColor(tcell.ColorDarkRed)
						} else {
							createButton.SetLabel(" [green]Draft Created Successfully! ")
							createButton.SetBackgroundColor(tcell.ColorDarkGreen)

							if selectedTemplate != nil {
								if selectedTemplate.Type == "initial" {
									targetPlan.IsDiscoveryDrafted = true
								} else if selectedTemplate.Type == "plan" {
									targetPlan.IsFinalPlanDrafted = true
								} else if selectedTemplate.Type == "follow_up" {
									targetPlan.IsFollowUpSent = true
								}
								// Update appState and persist
								appState.Assignments = assignments
								state.SaveState("state.json", appState)
								refreshTable()
							}
						}

						go func() {
							time.Sleep(3 * time.Second)
							app.QueueUpdateDraw(func() {
								createButton.SetLabel("Create Gmail Draft")
								createButton.SetBackgroundColor(tcell.ColorDefault)
							})
						}()
					})
				}()
			})

			for _, name := range names {
				tmplName := name
				templateList.AddItem(tmplName, "", 0, func() {
					updateDraftPreview(tmplName)
				})
			}

			templateList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
				updateDraftPreview(mainText)
			})

			if len(names) > 0 {
				updateDraftPreview(names[0])
			} else {
				previewText.SetText("\n\n(No templates found for this coach)")
			}

			detailBody.SwitchToPage("Drafting")
			app.SetFocus(templateList)
		})

		detailMenu.AddItem("", "", 0, nil) // Spacer at index 5

		detailMenu.AddItem("Back to Dashboard", "", 'q', func() {
			pages.SwitchToPage("Dashboard")
			app.SetFocus(table)
		})

		// Skip-logic for blank spacers (Index 1, 3, 5, 7)
		detailMenu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			cur := detailMenu.GetCurrentItem()
			if event.Key() == tcell.KeyDown {
				if cur == 0 {
					detailMenu.SetCurrentItem(2)
					return nil
				} else if cur == 2 {
					detailMenu.SetCurrentItem(4)
					return nil
				} else if cur == 4 {
					detailMenu.SetCurrentItem(6)
					return nil
				} else if cur == 6 {
					detailMenu.SetCurrentItem(8)
					return nil
				}
			} else if event.Key() == tcell.KeyUp {
				if cur == 8 {
					detailMenu.SetCurrentItem(6)
					return nil
				} else if cur == 6 {
					detailMenu.SetCurrentItem(4)
					return nil
				} else if cur == 4 {
					detailMenu.SetCurrentItem(2)
					return nil
				} else if cur == 2 {
					detailMenu.SetCurrentItem(0)
					return nil
				}
			}
			return event
		})

		// 3. Switch to Menu page
		pages.SwitchToPage("AssignmentDetail")
		detailBody.SwitchToPage("Menu")
		app.SetFocus(detailMenu)
	})

	detailTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			detailBody.SwitchToPage("Menu")
			app.SetFocus(detailMenu)
			return nil
		}
		return event
	})

	detailText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			detailBody.SwitchToPage("Menu")
			app.SetFocus(detailMenu)
			return nil
		}
		return event
	})

	coachScheduleTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			pages.SwitchToPage("Menu")
			app.SetFocus(list)
			return nil
		}
		return event
	})

	detailPage.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			// If we're on the Menu, go back to Dashboard.
			// If we're on Text/Table, the specific captures above will handle it.
			currentPage, _ := detailBody.GetFrontPage()
			if currentPage == "Menu" {
				pages.SwitchToPage("Dashboard")
				app.SetFocus(table)
			} else {
				detailBody.SwitchToPage("Menu")
				app.SetFocus(detailMenu)
			}
			return nil
		}
		return event
	})

	showExcelSheet := func(title, path, sheetName string) {
		rows, err := ingest.ReadRawExcel(path, sheetName)
		if err != nil {
			masterScheduleTable.Clear()
			masterScheduleTable.SetTitle(fmt.Sprintf(" %s (ESC/q: back) ", title))
			msg := fmt.Sprintf("  File not found or unreadable.\n  Path: %s\n\n  Please sync the spreadsheet first (s/r keys).", path)
			masterScheduleTable.SetCell(0, 0, tview.NewTableCell(msg).SetSelectable(true).SetExpansion(1))
			pages.SwitchToPage("MasterSchedule")
			app.SetFocus(masterScheduleTable)
			return
		}

		renderRows := len(rows)
		if renderRows == 0 {
			masterScheduleTable.Clear()
			masterScheduleTable.SetTitle(fmt.Sprintf(" %s (ESC/q: back) ", title))
			masterScheduleTable.SetCell(0, 0, tview.NewTableCell("  No entries found in this spreadsheet.").SetSelectable(true).SetExpansion(1))
			pages.SwitchToPage("MasterSchedule")
			app.SetFocus(masterScheduleTable)
			return
		}

		if renderRows > 500 {
			renderRows = 500
		}

		trueMaxColWidths := make(map[int]int)
		for r := 0; r < renderRows; r++ {
			for c, cell := range rows[r] {
				if len(cell) > trueMaxColWidths[c] {
					trueMaxColWidths[c] = len(cell)
				}
			}
		}

		masterScheduleTable.Clear()
		masterScheduleTable.SetTitle(fmt.Sprintf(" %s (ESC/q: back) ", title))
		for r := 0; r < renderRows; r++ {
			row := rows[r]
			for c, cell := range row {
				// Hybrid strategy: Cap column width at 30
				colWidth := trueMaxColWidths[c]
				if colWidth > 30 {
					colWidth = 30
				}

				display := cell
				if len(display) > colWidth {
					display = display[:colWidth-3] + "..."
				}
				padded := fmt.Sprintf(" %-*s ", colWidth, display)

				tableCell := tview.NewTableCell(padded)
				if r == 0 {
					tableCell.SetTextColor(tcell.ColorYellow)
				}
				masterScheduleTable.SetCell(r, c, tableCell)
			}
		}

		if len(rows) > 500 {
			footerRow := masterScheduleTable.GetRowCount()
			masterScheduleTable.SetCell(footerRow, 0, tview.NewTableCell(fmt.Sprintf("  ... (Truncated: %d rows total) ...", len(rows))).SetSelectable(false).SetTextColor(tcell.ColorDarkGray))
		}
		masterScheduleTable.Select(0, 0)
		masterScheduleTable.ScrollToBeginning()
		pages.SwitchToPage("MasterSchedule")
		app.SetFocus(masterScheduleTable)
	}

	masterScheduleTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			pages.SwitchToPage("Menu")
			app.SetFocus(list)
			return nil
		}
		return event
	})

	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			pages.SwitchToPage("Menu")
			app.SetFocus(list)
		}
	})

	showLogModal := func(title string) (*tview.TextView, func()) {
		var logView *tview.TextView
		logView = tview.NewTextView().
			SetDynamicColors(true).
			SetScrollable(true).
			SetChangedFunc(func() {
				logView.ScrollToEnd()
			})
		logView.SetBorder(true).SetTitle(" " + title + " ").SetTitleAlign(tview.AlignLeft)

		layout := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(logView, 15, 1, true).
				AddItem(nil, 0, 1, false), 80, 1, true).
			AddItem(nil, 0, 1, false)

		pages.AddPage("LogModal", layout, true, true)
		app.SetFocus(logView)

		return logView, func() {
			pages.RemovePage("LogModal")
			app.SetFocus(list)
		}
	}

	showEmailPicker := func(title, destFilename string, onComplete func(origName, query string)) {
		activeEmail := cfg.Coaches[cfg.ActiveCoach].GmailAccount

		form := tview.NewForm().
			AddTextView("Context:", fmt.Sprintf("Search for files in %s's inbox for emails matching these criteria:", activeEmail), 0, 2, false, false).
			AddInputField("Sender Name:", cfg.GmailDiscovery.SenderName, 40, nil, nil).
			AddInputField("Newer Than (Days):", fmt.Sprint(cfg.GmailDiscovery.NewerThanDays), 5, nil, nil)

		form.AddButton("Search", nil).
			AddButton("Cancel", func() {
				pages.RemovePage("DiscoverySettings")
				app.SetFocus(list)
			})
		form.SetBorder(true).SetTitle(fmt.Sprintf(" Search for %s... ", title)).SetTitleAlign(tview.AlignLeft)

		form.GetButton(0).SetSelectedFunc(func() {
			sender := form.GetFormItem(1).(*tview.InputField).GetText()
			daysStr := form.GetFormItem(2).(*tview.InputField).GetText()

			var days int
			fmt.Sscanf(daysStr, "%d", &days)
			if days <= 0 {
				days = 7
			}

			// Update and Persist Config
			cfg.GmailDiscovery.SenderName = sender
			cfg.GmailDiscovery.NewerThanDays = days
			_ = config.SaveConfig("config.json", cfg)

			pages.RemovePage("DiscoverySettings")

			// Proceed with original search logic
			logView, closeLog := showLogModal("Searching Gmail")
			logFn := func(msg string) {
				app.QueueUpdateDraw(func() {
					fmt.Fprintf(logView, "[-] %s\n", msg)
				})
			}

			go func() {
				logFn(fmt.Sprintf("Searching for emails from '%s' (last %d days)...", sender, days))
				query := fmt.Sprintf("from:%s has:attachment newer_than:%dd", sender, days)
				messages, err := driveSvc.SearchMessages(query)

				app.QueueUpdateDraw(func() {
					if err != nil || len(messages) == 0 {
						if err != nil {
							fmt.Fprintf(logView, "\n[red]Error searching Gmail: %v[-]\n", err)
						} else {
							fmt.Fprintf(logView, "\n[red]No matching emails found.[-]\n")
						}
						fmt.Fprintf(logView, "\nPress q or ESC to return.")
						logView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
							if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
								closeLog()
							}
							return event
						})
						return
					}

					// If messages found, show a picker list
					pickerList := tview.NewList().ShowSecondaryText(true)
					pickerList.SetBorder(true).SetTitle(" Select Email to Sync ").SetTitleAlign(tview.AlignLeft)

					for _, msg := range messages {
						// Use a shortened snippet or date as secondary text
						pickerList.AddItem(msg.Subject, msg.Date, 0, nil)
					}

					pickerList.AddItem("Cancel", "Return to main menu", 'q', func() {
						pages.RemovePage("EmailPicker")
						app.SetFocus(list)
					})

					pickerList.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
						if mainText == "Cancel" {
							return
						}

						selectedMsg := messages[index]
						pages.RemovePage("EmailPicker")

						// Now start the download process
						logView, closeLog := showLogModal("Downloading Attachment")
						logFn := func(msg string) {
							app.QueueUpdateDraw(func() {
								fmt.Fprintf(logView, "[-] %s\n", msg)
							})
						}

						go func() {
							logFn(fmt.Sprintf("Inspecting email: '%s'...", selectedMsg.Subject))
							attachments, err := driveSvc.GetMessageAttachments(selectedMsg.ID)
							if err != nil {
								app.QueueUpdateDraw(func() {
									fmt.Fprintf(logView, "\n[red]Failed to get attachments: %v[-]\n\nPress q or ESC to return.", err)
									logView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
										if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
											closeLog()
										}
										return event
									})
								})
								return
							}

							var targetAttachID string
							var targetFilename string
							for _, a := range attachments {
								ext := strings.ToLower(filepath.Ext(a.Filename))
								if ext == ".xlsx" || ext == ".xls" {
									targetAttachID = a.ID
									targetFilename = a.Filename
									break
								}
							}

							if targetAttachID == "" {
								app.QueueUpdateDraw(func() {
									fmt.Fprintf(logView, "\n[red]No spreadsheet attachment found in this email.[-]\n\nPress q or ESC to return.")
									logView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
										if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
											closeLog()
										}
										return event
									})
								})
								return
							}

							logFn(fmt.Sprintf("Found spreadsheet: '%s'. Downloading...", targetFilename))
							err = driveSvc.DownloadAttachment(selectedMsg.ID, targetAttachID, destFilename)
							app.QueueUpdateDraw(func() {
								if err != nil {
									fmt.Fprintf(logView, "\n[red]Download failed: %v[-]\n\nPress q or ESC.", err)
								} else {
									fmt.Fprintf(logView, "\n[green]SUCCESS! Saved to artifacts/ as %s[-]\n\nPress q or ESC to close.", destFilename)
									onComplete(targetFilename, selectedMsg.Subject)
								}
								logView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
									if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
										closeLog()
									}
									return event
								})
							})
						}()
					})

					modalPicker := tview.NewFlex().
						AddItem(nil, 0, 1, false).
						AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
							AddItem(nil, 0, 1, false).
							AddItem(pickerList, 15, 1, true).
							AddItem(nil, 0, 1, false), 80, 1, true).
						AddItem(nil, 0, 1, false)

					closeLog() // Remove the "Searching" log
					pages.AddPage("EmailPicker", modalPicker, true, true)
					app.SetFocus(pickerList)
				})
			}()
		})

		modalForm := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(form, 12, 1, true).
				AddItem(nil, 0, 1, false), 80, 1, true).
			AddItem(nil, 0, 1, false)

		pages.AddPage("DiscoverySettings", modalForm, true, true)
		app.SetFocus(form)
	}

	checkFolderStatus := func() {
		if len(assignments) == 0 {
			return
		}
		logView, closeLog := showLogModal("Checking Drive Folders")
		logFn := func(msg string) {
			app.QueueUpdateDraw(func() {
				fmt.Fprintf(logView, "[-] %s\n", msg)
			})
		}

		go func() {
			logFn("Scanning solo student folders...")
			soloItems, err1 := driveSvc.ListFolderContents(cfg.Drive.WorkshopParentFolderID)
			if err1 != nil {
				logFn(fmt.Sprintf("[red]Warning: solo folder scan failed: %v[-]", err1))
			}
			logFn(fmt.Sprintf("Found %d items in solo folder", len(soloItems)))

			logFn("Scanning team group folders...")
			teamItems, err2 := driveSvc.ListFolderContents(cfg.Drive.TeamsFolderID)
			if err2 != nil {
				logFn(fmt.Sprintf("[red]Warning: team folder scan failed: %v[-]", err2))
			}
			logFn(fmt.Sprintf("Found %d items in teams folder", len(teamItems)))

			allItems := append(soloItems, teamItems...)

			matched := 0
			for i := range assignments {
				if assignments[i].IsReserved {
					continue
				}
				for _, item := range allItems {
					if domain.FolderMatchesAssignment(item.Name, assignments[i].SubjectName) {
						assignments[i].IsWorkspaceCreated = true
						logFn(fmt.Sprintf("  \u2705 '%s' matched folder '%s'", assignments[i].SubjectName, item.Name))
						matched++

						// Check inside the folder for a training plan
						if item.ID != "" {
							contents, err := driveSvc.ListFolderContents(item.ID)
							if err == nil {
								for _, child := range contents {
									if strings.Contains(strings.ToLower(child.Name), "plan") ||
										strings.Contains(strings.ToLower(child.Name), "worksheet") ||
										strings.Contains(strings.ToLower(child.Name), "training") {
										assignments[i].HasTrainingPlan = true
										assignments[i].DriveFileID = child.ID
										logFn(fmt.Sprintf("    \U0001f4c4 Found plan: '%s' (ID: %s)", child.Name, child.ID))
										break
									}
								}
								if !assignments[i].HasTrainingPlan {
									logFn("    \u274c No training plan found inside folder")
								}
							}
						}
						break
					}
				}
				if !assignments[i].IsWorkspaceCreated {
					logFn(fmt.Sprintf("  \u274c '%s' \u2014 no matching folder", assignments[i].SubjectName))
				}
			}

			app.QueueUpdateDraw(func() {
				folderStatusChecked = true
				refreshTable()

				// Persist the found IDs and statuses
				appState.Assignments = assignments
				state.SaveState("state.json", appState)

				fmt.Fprintf(logView, "\n[green]Done! %d/%d assignments have folders.[-]\n\nPress q or ESC to close.", matched, len(assignments))
				logView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
						closeLog()
						app.SetFocus(table)
					}
					return event
				})
			})
		}()
	}

	// masterScheduleTable wraparound removal

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("Menu")
			app.SetFocus(list)
			return nil
		}
		if event.Rune() == 'r' {
			checkFolderStatus()
			return nil
		}
		if event.Key() == tcell.KeyDown {
			row, _ := table.GetSelection()
			rowCount := table.GetRowCount()

			// Safety: Check if there are ANY assignment rows to wrap around to
			hasAny := false
			for r := 0; r < rowCount; r++ {
				if isRowReference(table, r, 0, "assignment") {
					hasAny = true
					break
				}
			}
			if !hasAny {
				return event
			}

			isLast := true
			for r := row + 1; r < rowCount; r++ {
				if isRowReference(table, r, 0, "assignment") {
					isLast = false
					break
				}
			}
			if isLast {
				for r := 0; r < rowCount; r++ {
					if isRowReference(table, r, 0, "assignment") {
						table.Select(r, 0)
						return nil
					}
				}
			}
		}
		if event.Key() == tcell.KeyUp {
			row, _ := table.GetSelection()
			rowCount := table.GetRowCount()

			// Safety: Check if there are ANY assignment rows to wrap around to
			hasAny := false
			for r := 0; r < rowCount; r++ {
				if isRowReference(table, r, 0, "assignment") {
					hasAny = true
					break
				}
			}
			if !hasAny {
				return event
			}

			isFirst := true
			for r := row - 1; r >= 0; r-- {
				if isRowReference(table, r, 0, "assignment") {
					isFirst = false
					break
				}
			}
			if isFirst {
				for r := rowCount - 1; r >= 0; r-- {
					if isRowReference(table, r, 0, "assignment") {
						table.Select(r, 0)
						return nil
					}
				}
			}
		}
		return event
	})

	showCoachSelection := func() {
		coachList := tview.NewList().ShowSecondaryText(false)
		coachList.SetBorder(true).SetTitle(" Select Active Coach ").SetTitleAlign(tview.AlignLeft)

		// Sort coach names for stable ordering
		var keys []string
		for key := range cfg.Coaches {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			coachKey := key
			profile := cfg.Coaches[coachKey]
			label := fmt.Sprintf("%s (%s)", profile.Name, profile.DraftedFrom)
			if coachKey == cfg.ActiveCoach {
				label = "* " + label
			}
			coachList.AddItem(label, "", rune(0), func() {
				cfg.ActiveCoach = coachKey
				_ = config.SaveConfig("config.json", cfg)
				loadAssignments()
				folderStatusChecked = false
				mainContainer.SetTitle(fmt.Sprintf(" COACH ASSIST %s - Active Profile: %s ", version, cfg.Coaches[coachKey].Name))
				pages.RemovePage("CoachSelection")
				app.SetFocus(list)
			})
		}

		coachList.AddItem("Cancel", "", 'q', func() {
			pages.RemovePage("CoachSelection")
			app.SetFocus(list)
		})

		modalFlex := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(coachList, len(cfg.Coaches)+4, 1, true).
				AddItem(nil, 0, 1, false), 50, 1, true).
			AddItem(nil, 0, 1, false)

		pages.AddPage("CoachSelection", modalFlex, true, true)
		app.SetFocus(coachList)
	}

	list.AddItem("👤 Select Active Coach Profile", "", 'c', showCoachSelection)

	list.AddItem("⚠️ Reset Local Workspace", "", 'x', func() {
		confirmModal := tview.NewModal().
			SetText("Are you absolutely sure you want to purge all local assignments?\n(This will delete state.json, latest-schedule.xlsx, and latest-roster.xlsx)").
			AddButtons([]string{"Purge", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				pages.RemovePage("ConfirmModal")
				app.SetFocus(list)
				if buttonLabel == "Purge" {
					// Perform targeted state wipe (keep original downloads)
					os.Remove("state.json")
					os.Remove(filepath.Join("artifacts", "latest-schedule.xlsx"))
					os.Remove(filepath.Join("artifacts", "latest-roster.xlsx"))
					appState = state.AppState{} // Erase memory
					assignments = nil           // Erase RAM slices
					allScheduleRows = nil       // Flush master schedule data
					folderStatusChecked = false

					for i := 0; i < list.GetItemCount(); i++ {
						main, _ := list.GetItemText(i)
						if strings.Contains(main, "Sync Master Schedule") {
							list.SetItemText(i, "[ ] Sync Master Schedule", fmt.Sprintf("Select latest .xlsx from %s", cfg.GmailDiscovery.SenderName))
						} else if strings.Contains(main, "Sync 'Who Makes Dives' Roster") {
							list.SetItemText(i, "[ ] Sync 'Who Makes Dives' Roster", fmt.Sprintf("Select latest .xlsx from %s", cfg.GmailDiscovery.SenderName))
						}
					}
				}
			})
		pages.AddPage("ConfirmModal", confirmModal, true, true)
		app.SetFocus(confirmModal)
	})

	list.AddItem("Sync Master Schedule", fmt.Sprintf("Select latest .xlsx from %s", cfg.GmailDiscovery.SenderName), 's', func() {
		showEmailPicker("Schedule", "latest-schedule.xlsx", func(origName, query string) {
			appState.LastScheduleSubject = query
			appState.LastScheduleFilename = origName
			_ = state.SaveState("state.json", appState)

			list.SetItemText(2, "[✅] Sync Master Schedule", fmt.Sprintf("File: %s | Subject: '%s'", origName, query))
			loadAssignments()
		})
	})

	list.AddItem("Sync 'Who Makes Dives' Roster", fmt.Sprintf("Select latest .xlsx from %s", cfg.GmailDiscovery.SenderName), 'r', func() {
		showEmailPicker("Roster", "latest-roster.xlsx", func(origName, query string) {
			appState.LastRosterSubject = query
			appState.LastRosterFilename = origName
			_ = state.SaveState("state.json", appState)

			list.SetItemText(3, "[✅] Sync 'Who Makes Dives' Roster", fmt.Sprintf("File: %s | Subject: '%s'", origName, query))
			loadAssignments()
		})
	})

	list.AddItem("View Pending Assignments", "", 'a', func() {
		refreshTable()
		pages.SwitchToPage("Dashboard")
		app.SetFocus(table)
	})

	list.AddItem("View Current Coach's Schedule", "", 'v', func() {
		coachName := cfg.Coaches[cfg.ActiveCoach].Name
		
		coachScheduleTable.SetTitle(fmt.Sprintf(" Schedule for Coach: %s (ESC/q to back) ", coachName))
		coachScheduleTable.Clear()
		coachScheduleTable.SetCell(0, 0, tview.NewTableCell("Date").SetTextColor(tcell.ColorYellow))
		coachScheduleTable.SetCell(0, 1, tview.NewTableCell("Time").SetTextColor(tcell.ColorYellow))
		coachScheduleTable.SetCell(0, 2, tview.NewTableCell("Coach 1").SetTextColor(tcell.ColorYellow))
		coachScheduleTable.SetCell(0, 3, tview.NewTableCell("Group 1").SetTextColor(tcell.ColorYellow))
		coachScheduleTable.SetCell(0, 4, tview.NewTableCell("Coach 2").SetTextColor(tcell.ColorYellow))
		coachScheduleTable.SetCell(0, 5, tview.NewTableCell("Group 2").SetTextColor(tcell.ColorYellow))

		rowIdx := 1
		for _, sched := range allScheduleRows {
			if strings.Contains(sched.Coach1, coachName) || strings.Contains(sched.Coach2, coachName) {
				coachScheduleTable.SetCell(rowIdx, 0, tview.NewTableCell(sched.Date).SetReference("data"))
				coachScheduleTable.SetCell(rowIdx, 1, tview.NewTableCell(sched.FlyingAt))
				coachScheduleTable.SetCell(rowIdx, 2, tview.NewTableCell(sched.Coach1))
				coachScheduleTable.SetCell(rowIdx, 3, tview.NewTableCell(sched.Group1))
				coachScheduleTable.SetCell(rowIdx, 4, tview.NewTableCell(sched.Coach2))
				coachScheduleTable.SetCell(rowIdx, 5, tview.NewTableCell(sched.Group2))
				rowIdx++
			}
		}
		if rowIdx == 1 {
			coachScheduleTable.SetCell(1, 0, tview.NewTableCell("No schedule entries found for you.").SetSelectable(true).SetExpansion(1))
			coachScheduleTable.Select(1, 0)
		} else {
			coachScheduleTable.Select(1, 0)
		}
		coachScheduleTable.ScrollToBeginning()
		pages.SwitchToPage("CoachSchedule")
		app.SetFocus(coachScheduleTable)
	})

	list.AddItem("View Master Schedule", "", 'm', func() {
		schedPath := filepath.Join("artifacts", "latest-schedule.xlsx")
		showExcelSheet("Master Schedule", schedPath, "")
	})

	list.AddItem("View 'Who Makes Dives' Roster", "", 'w', func() {
		rosterPath := filepath.Join("artifacts", "latest-roster.xlsx")
		showExcelSheet("Who Makes Dives", rosterPath, "Who makes dives")
	})

	list.AddItem("View Student Email List", "", 'e', func() {
		rosterPath := filepath.Join("artifacts", "latest-roster.xlsx")
		showExcelSheet("Student Email List", rosterPath, "Student list")
	})

	list.AddItem("Quit", "", 'q', func() {
		app.Stop()
	})

	// === BOOTUP: Check for existing artifacts ===
	schedPath := filepath.Join("artifacts", "latest-schedule.xlsx")
	if _, err := os.Stat(schedPath); err == nil {
		subj := appState.LastScheduleSubject
		fn := appState.LastScheduleFilename
		if subj == "" {
			subj = "unknown"
		}
		if fn == "" {
			fn = "latest-schedule.xlsx"
		}
		list.SetItemText(2, "[✅] Sync Master Schedule", fmt.Sprintf("File: %s | Subject: '%s'", fn, subj))
	}

	rosterPath := filepath.Join("artifacts", "latest-roster.xlsx")
	if _, err := os.Stat(rosterPath); err == nil {
		subj := appState.LastRosterSubject
		fn := appState.LastRosterFilename
		if subj == "" {
			subj = "unknown"
		}
		if fn == "" {
			fn = "latest-roster.xlsx"
		}
		list.SetItemText(3, "[✅] Sync 'Who Makes Dives' Roster", fmt.Sprintf("File: %s | Subject: '%s'", fn, subj))
	}

	pages.AddPage("Menu", list, true, true)
	pages.AddPage("Dashboard", table, true, false)
	pages.AddPage("AssignmentDetail", detailPage, true, false)
	pages.AddPage("CoachSchedule", coachScheduleTable, true, false)
	pages.AddPage("MasterSchedule", masterScheduleTable, true, false)

	mainContainer.AddItem(pages, 0, 1, true)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			os.Exit(0) // Aggressive kill to handle UI freezes
			return nil
		}
		return event
	})

	if err := app.SetRoot(mainContainer, true).EnableMouse(false).Run(); err != nil {
		panic(err)
	}
}

func colNumToLetter(n int) string {
	var s string
	for n > 0 {
		n--
		s = string(rune('A'+(n%26))) + s
		n /= 26
	}
	return s
}

func getTargetCol(merges []drive.SheetMerge, row, col int) int {
	for _, m := range merges {
		if m.StartRowIndex <= row && row < m.EndRowIndex {
			if m.StartColumnIndex <= col && col < m.EndColumnIndex {
				return m.EndColumnIndex
			}
		}
	}
	return col + 1
}
