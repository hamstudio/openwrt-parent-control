import SwiftUI
import ParentControlCore

struct MemberEditorView: View {
    @Environment(\.dismiss) var dismiss
    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared

    let editingMember: Member?

    @State private var name: String = ""
    @State private var avatar: String = "boy"
    @State private var selectedMACs: [String] = []
    @State private var quotaMinutes: Double = 120
    
    // Multi-time-range and action modes
    @State private var scheduleEnabled: Bool = true
    @State private var scheduleAction: String = "block" // "block" or "allow"
    @State private var selectedDays: Set<Int> = [0, 1, 2, 3, 4, 5, 6]
    @State private var timeRanges: [TimeRangeItem] = [
        TimeRangeItem(start: Calendar.current.date(from: DateComponents(hour: 21, minute: 30)) ?? Date(),
                      end: Calendar.current.date(from: DateComponents(hour: 7, minute: 0)) ?? Date())
    ]
    
    @State private var blockedAppIDs: [Int] = []
    @State private var isSaving: Bool = false
    @State private var showingDeleteAlert: Bool = false

    struct TimeRangeItem: Identifiable {
        let id = UUID()
        var start: Date
        var end: Date
    }

    init(member: Member?) {
        self.editingMember = member
    }

    var body: some View {
        NavigationStack {
            Form {
                // Basic Information
                Section(i18n.t("memberBasicSection")) {
                    TextField(i18n.t("memberNamePlaceholder"), text: $name)

                    Picker(i18n.t("avatarLabel"), selection: $avatar) {
                        Text(i18n.t("avatarBoy")).tag("boy")
                        Text(i18n.t("avatarGirl")).tag("girl")
                        Text(i18n.t("avatarStudent")).tag("student")
                        Text(i18n.t("avatarChild")).tag("child")
                    }
                }

                // Device Binding
                Section("\(i18n.t("deviceBindingSection")) (\(selectedMACs.count))") {
                    if appState.devices.isEmpty {
                        Text(i18n.t("noDevices"))
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(appState.devices) { device in
                            let isChecked = selectedMACs.contains(device.mac)
                            Button {
                                toggleDevice(device.mac)
                                HapticManager.impact(.light)
                            } label: {
                                HStack {
                                    Image(systemName: isChecked ? "checkmark.circle.fill" : "circle")
                                        .foregroundColor(isChecked ? .guardianGreen : .secondary)

                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(device.displayName)
                                            .font(.subheadline)
                                            .foregroundColor(.primary)
                                        Text("\(device.ip) · \(device.vendor)")
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }

                                    Spacer()

                                    if device.online {
                                        Circle()
                                            .fill(Color.guardianGreen)
                                            .frame(width: 6, height: 6)
                                    }
                                }
                            }
                        }
                    }
                }

                // Daily Quota
                Section(i18n.t("dailyQuotaSection")) {
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text(i18n.t("quotaDuration"))
                            Spacer()
                            Text(quotaMinutes == 0 ? i18n.t("unlimited") : "\(Int(quotaMinutes)) \(i18n.t("minutes"))")
                                .bold()
                                .foregroundColor(.guardianGreen)
                        }

                        Slider(value: $quotaMinutes, in: 0...360, step: 15)
                            .tint(.guardianGreen)
                    }
                }

                // Schedule Restrictions
                Section(i18n.t("scheduleSection")) {
                    Toggle(i18n.t("scheduleSwitch"), isOn: $scheduleEnabled)
                        .tint(.guardianGreen)

                    if scheduleEnabled {
                        Picker("Mode", selection: $scheduleAction) {
                            Text("🚫 \(i18n.t("scheduleActionBlock"))").tag("block")
                            Text("✅ \(i18n.t("scheduleActionAllow"))").tag("allow")
                        }
                        .pickerStyle(.segmented)

                        // Active Days Selector
                        VStack(alignment: .leading, spacing: 8) {
                            Text(i18n.t("repeatDays"))
                                .font(.caption)
                                .foregroundColor(.secondary)

                            HStack(spacing: 6) {
                                ForEach(0..<7) { day in
                                    let isSelected = selectedDays.contains(day)
                                    Button {
                                        if isSelected {
                                            selectedDays.remove(day)
                                        } else {
                                            selectedDays.insert(day)
                                        }
                                        HapticManager.impact(.light)
                                    } label: {
                                        Text(dayShortName(day))
                                            .font(.caption.bold())
                                            .frame(maxWidth: .infinity)
                                            .padding(.vertical, 8)
                                            .background(isSelected ? Color.guardianGreen : Color.adaptiveGray5)
                                            .foregroundColor(isSelected ? .white : .primary)
                                            .cornerRadius(8)
                                    }
                                    .buttonStyle(.borderless)
                                }
                            }
                        }
                        .padding(.vertical, 4)

                        // Time Range List
                        VStack(alignment: .leading, spacing: 10) {
                            HStack {
                                Text("Time Ranges (\(timeRanges.count))")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                Spacer()
                                Button {
                                    timeRanges.append(TimeRangeItem(start: Date(), end: Date().addingTimeInterval(3600)))
                                    HapticManager.impact(.light)
                                } label: {
                                    Label("Add Range", systemImage: "plus")
                                        .font(.caption.bold())
                                        .foregroundColor(.guardianGreen)
                                }
                                .buttonStyle(.borderless)
                            }

                            ForEach($timeRanges) { $range in
                                HStack(spacing: 8) {
                                    DatePicker("", selection: $range.start, displayedComponents: .hourAndMinute)
                                        .labelsHidden()
                                    Text("to")
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    DatePicker("", selection: $range.end, displayedComponents: .hourAndMinute)
                                        .labelsHidden()
                                    
                                    Spacer()

                                    if timeRanges.count > 1 {
                                        Button {
                                            timeRanges.removeAll { $0.id == range.id }
                                            HapticManager.impact(.light)
                                        } label: {
                                            Image(systemName: "trash")
                                                .font(.caption)
                                                .foregroundColor(.red)
                                        }
                                        .buttonStyle(.borderless)
                                    }
                                }
                                .padding(8)
                                .background(Color.adaptiveSecondaryBackground)
                                .cornerRadius(10)
                            }
                        }
                    }
                }

                // L7 DPI App Restrictions
                Section(i18n.t("blockedAppsSection")) {
                    AppSelectorView(
                        selectedAppIDs: $blockedAppIDs,
                        categories: appState.categories
                    )
                }

                // Delete Button (Edit Mode Only)
                if let member = editingMember {
                    Section {
                        Button(role: .destructive) {
                            showingDeleteAlert = true
                        } label: {
                            HStack {
                                Spacer()
                                Text(i18n.t("btnDelete"))
                                Spacer()
                            }
                        }
                    }
                }
            }
            .navigationTitle(editingMember == nil ? i18n.t("addMemberTitle") : i18n.t("editMemberTitle"))
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(i18n.t("cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(i18n.t("btnSave")) {
                        saveMember()
                    }
                    .bold()
                    .foregroundColor(.guardianGreen)
                    .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty || isSaving)
                }
            }
            .alert(i18n.t("btnDelete"), isPresented: $showingDeleteAlert) {
                Button(i18n.t("btnDelete"), role: .destructive) {
                    if let m = editingMember {
                        HapticManager.notification(.warning)
                        Task {
                            await appState.deleteMember(id: m.id)
                            dismiss()
                        }
                    }
                }
                Button(i18n.t("cancel"), role: .cancel) {}
            } message: {
                Text(i18n.t("deleteConfirm"))
            }
            .onAppear {
                loadInitialData()
            }
        }
    }

    private func dayShortName(_ day: Int) -> String {
        switch day {
        case 0: return "Sun"
        case 1: return "Mon"
        case 2: return "Tue"
        case 3: return "Wed"
        case 4: return "Thu"
        case 5: return "Fri"
        case 6: return "Sat"
        default: return ""
        }
    }

    private func toggleDevice(_ mac: String) {
        if let idx = selectedMACs.firstIndex(of: mac) {
            selectedMACs.remove(at: idx)
        } else {
            selectedMACs.append(mac)
        }
    }

    private func loadInitialData() {
        if let m = editingMember {
            name = m.name
            avatar = m.avatar
            selectedMACs = m.deviceMACs
            quotaMinutes = Double(m.quotaMinutes)
            blockedAppIDs = m.blockedAppIDs
            scheduleEnabled = m.schedule.enabled
            scheduleAction = m.schedule.action.isEmpty ? "block" : m.schedule.action
            if !m.schedule.days.isEmpty {
                selectedDays = Set(m.schedule.days)
            }

            let formatter = DateFormatter()
            formatter.dateFormat = "HH:mm"

            if !m.schedule.timeRanges.isEmpty {
                timeRanges = m.schedule.timeRanges.map { tr in
                    let s = formatter.date(from: tr.startTime) ?? Date()
                    let e = formatter.date(from: tr.endTime) ?? Date()
                    return TimeRangeItem(start: s, end: e)
                }
            }
        }
    }

    private func saveMember() {
        isSaving = true
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"

        let ranges = timeRanges.map {
            TimeRange(startTime: formatter.string(from: $0.start),
                      endTime: formatter.string(from: $0.end))
        }

        let member = Member(
            id: editingMember?.id ?? "m_\(Int(Date().timeIntervalSince1970 * 1000))",
            name: name.trimmingCharacters(in: .whitespaces),
            avatar: avatar,
            deviceMACs: selectedMACs,
            enabled: true,
            isLocked: editingMember?.isLocked ?? false,
            bonusUntil: editingMember?.bonusUntil,
            quotaMinutes: Int(quotaMinutes),
            usedMinutes: editingMember?.usedMinutes ?? 0,
            schedule: ScheduleRule(
                enabled: scheduleEnabled && !ranges.isEmpty,
                days: Array(selectedDays).sorted(),
                timeRanges: ranges,
                action: scheduleAction
            ),
            blockedAppIDs: blockedAppIDs,
            safeSearch: true,
            blockAdult: true
        )

        Task {
            let success = await appState.saveMember(member)
            if success {
                HapticManager.notification(.success)
                dismiss()
            }
            isSaving = false
        }
    }
}
