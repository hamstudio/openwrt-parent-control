import SwiftUI
import ParentControlCore

struct MemberEditorView: View {
    @Environment(\.dismiss) var dismiss
    @EnvironmentObject var appState: AppState

    let editingMember: Member?

    @State private var name: String = ""
    @State private var avatar: String = "boy"
    @State private var selectedMACs: [String] = []
    @State private var quotaMinutes: Double = 120
    
    // 多时间段与动作模式
    @State private var scheduleEnabled: Bool = true
    @State private var scheduleAction: String = "block" // "block" or "allow"
    @State private var selectedDays: Set<Int> = [0, 1, 2, 3, 4, 5, 6]
    @State private var timeRanges: [TimeRangeItem] = [
        TimeRangeItem(start: Calendar.current.date(from: DateComponents(hour: 21, minute: 30)) ?? Date(),
                      end: Calendar.current.date(from: DateComponents(hour: 7, minute: 0)) ?? Date())
    ]
    
    @State private var blockedAppIDs: [Int] = []
    @State private var isSaving: Bool = false

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
                // 基本信息
                Section("成员基本信息") {
                    TextField("例如：小明、大宝", text: $name)

                    Picker("头像标识", selection: $avatar) {
                        Text("👦 男孩").tag("boy")
                        Text("👧 女孩").tag("girl")
                        Text("🧑‍🎓 学生").tag("student")
                        Text("👶 儿童").tag("child")
                    }
                }

                // 设备绑定
                Section("绑定局域网设备 (\(selectedMACs.count) 台)") {
                    if appState.devices.isEmpty {
                        Text("未探测到局域网设备")
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

                // 每日配额
                Section("每日活跃上网限额") {
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Text("配额时长")
                            Spacer()
                            Text(quotaMinutes == 0 ? "不限时" : "\(Int(quotaMinutes)) 分钟")
                                .bold()
                                .foregroundColor(.guardianGreen)
                        }

                        Slider(value: $quotaMinutes, in: 0...360, step: 15)
                            .tint(.guardianGreen)
                    }
                }

                // 多时间段计划
                Section("健康上网时间段管控") {
                    Toggle("启用时间段管控计划", isOn: $scheduleEnabled)
                        .tint(.guardianGreen)

                    if scheduleEnabled {
                        Picker("管控模式", selection: $scheduleAction) {
                            Text("🚫 设为禁网时段").tag("block")
                            Text("✅ 仅在时段内允许").tag("allow")
                        }
                        .pickerStyle(.segmented)

                        // 快捷生效日期
                        VStack(alignment: .leading, spacing: 8) {
                            Text("生效星期")
                                .font(.caption)
                                .foregroundColor(.secondary)

                            HStack(spacing: 6) {
                                ForEach(0..<7) { day in
                                    let isSelected = selectedDays.contains(day)
                                    Button {
                                        if isSelected {
                                            if selectedDays.count > 1 { selectedDays.remove(day) }
                                        } else {
                                            selectedDays.insert(day)
                                        }
                                    } label: {
                                        Text(dayShortName(day))
                                            .font(.caption.bold())
                                            .frame(maxWidth: .infinity)
                                            .padding(.vertical, 6)
                                            .background(isSelected ? Color.guardianGreen : Color.adaptiveGray5)
                                            .foregroundColor(isSelected ? .white : .primary)
                                            .cornerRadius(8)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                        }

                        // 多时间段列表
                        VStack(alignment: .leading, spacing: 10) {
                            HStack {
                                Text("管控时间段列表 (\(timeRanges.count) 个)")
                                    .font(.caption)
                                    .foregroundColor(.secondary)
                                Spacer()
                                Button {
                                    timeRanges.append(TimeRangeItem(
                                        start: Calendar.current.date(from: DateComponents(hour: 12, minute: 0)) ?? Date(),
                                        end: Calendar.current.date(from: DateComponents(hour: 14, minute: 0)) ?? Date()
                                    ))
                                } label: {
                                    Label("添加时段", systemImage: "plus.circle.fill")
                                        .font(.caption.bold())
                                        .foregroundColor(.guardianGreen)
                                }
                            }

                            ForEach(timeRanges.indices, id: \.self) { idx in
                                HStack(spacing: 8) {
                                    Text("时段 \(idx + 1)")
                                        .font(.caption2.bold())
                                        .foregroundColor(.secondary)

                                    DatePicker("", selection: $timeRanges[idx].start, displayedComponents: .hourAndMinute)
                                        .labelsHidden()

                                    Text("至")
                                        .font(.caption)
                                        .foregroundColor(.secondary)

                                    DatePicker("", selection: $timeRanges[idx].end, displayedComponents: .hourAndMinute)
                                        .labelsHidden()

                                    if timeRanges.count > 1 {
                                        Button {
                                            timeRanges.remove(at: idx)
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

                // 深度 L7 DPI 应用封禁
                Section("限制应用层访问管控") {
                    AppSelectorView(
                        selectedAppIDs: $blockedAppIDs,
                        categories: appState.categories
                    )
                }

                // 删除按钮 (仅编辑模式)
                if let member = editingMember {
                    Section {
                        Button(role: .destructive) {
                            HapticManager.notification(.warning)
                            Task {
                                await appState.deleteMember(id: member.id)
                                dismiss()
                            }
                        } label: {
                            HStack {
                                Spacer()
                                Text("删除该受管成员")
                                Spacer()
                            }
                        }
                    }
                }
            }
            .navigationTitle(editingMember == nil ? "添加受管成员" : "编辑规则")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("保存") {
                        saveMember()
                    }
                    .bold()
                    .foregroundColor(.guardianGreen)
                    .disabled(name.trimmingCharacters(in: .whitespaces).isEmpty || isSaving)
                }
            }
            .onAppear {
                loadInitialData()
            }
        }
    }

    private func dayShortName(_ day: Int) -> String {
        switch day {
        case 0: return "日"
        case 1: return "一"
        case 2: return "二"
        case 3: return "三"
        case 4: return "四"
        case 5: return "五"
        case 6: return "六"
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
