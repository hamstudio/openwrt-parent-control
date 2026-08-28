import SwiftUI
import ParentControlCore

enum DeviceFilter: String, CaseIterable, Identifiable {
    case all
    case online
    case locked
    case unassigned

    var id: String { rawValue }

    func title(i18n: I18n) -> String {
        switch self {
        case .all: return i18n.t("filterAll")
        case .online: return i18n.t("filterOnline")
        case .locked: return i18n.t("filterLocked")
        case .unassigned: return i18n.t("filterUnassigned")
        }
    }
}

struct DevicesListView: View {
    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared

    @State private var searchText: String = ""
    @State private var selectedFilter: DeviceFilter = .all
    @State private var assigningDevice: Device?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Filter Picker
                Picker("Filter", selection: $selectedFilter) {
                    ForEach(DeviceFilter.allCases) { filter in
                        Text(filter.title(i18n: i18n)).tag(filter)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.vertical, 8)
                .background(Color.adaptiveBackground)

                // Device List
                List {
                    Section {
                        HStack {
                            Text("\(i18n.t("statDevices")): \(onlineCount) / \(appState.devices.count)")
                                .font(.caption.bold())
                                .foregroundColor(.secondary)
                            Spacer()
                            if appState.isRefreshing {
                                ProgressView()
                                    .scaleEffect(0.8)
                            }
                        }
                    }

                    if filteredDevices.isEmpty {
                        Section {
                            VStack(spacing: 12) {
                                Image(systemName: "laptopcomputer.and.iphone")
                                    .font(.system(size: 44))
                                    .foregroundColor(.secondary)
                                Text(i18n.t("noDevices"))
                                    .font(.subheadline)
                                    .foregroundColor(.secondary)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 32)
                        }
                    } else {
                        Section {
                            ForEach(filteredDevices) { device in
                                DeviceRowView(device: device) {
                                    assigningDevice = device
                                }
                            }
                        }
                    }
                }
            }
            .background(Color.adaptiveGroupedBackground)
            .navigationTitle(i18n.t("devicesTitle"))
            .searchable(text: $searchText, prompt: i18n.t("searchDevicePlaceholder"))
            .refreshable {
                await appState.refreshAll()
            }
            .sheet(item: $assigningDevice) { dev in
                DeviceAssignSheet(device: dev)
            }
        }
    }

    private var onlineCount: Int {
        appState.devices.filter { $0.online }.count
    }

    private var filteredDevices: [Device] {
        appState.devices.filter { d in
            // Search filter
            let matchesSearch = searchText.isEmpty ||
                d.hostname.localizedCaseInsensitiveContains(searchText) ||
                d.ip.contains(searchText) ||
                d.mac.localizedCaseInsensitiveContains(searchText) ||
                d.vendor.localizedCaseInsensitiveContains(searchText)

            guard matchesSearch else { return false }

            // Category filter
            switch selectedFilter {
            case .all:
                return true
            case .online:
                return d.online
            case .locked:
                return d.isLocked == true
            case .unassigned:
                let isAssigned = appState.members.contains { m in
                    m.id == d.memberId || m.deviceMACs.contains(d.mac)
                }
                return !isAssigned
            }
        }
    }
}

struct DeviceRowView: View {
    let device: Device
    let onAssign: () -> Void

    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared

    var body: some View {
        VStack(spacing: 10) {
            HStack(spacing: 12) {
                // Status Indicator and Vendor Icon
                ZStack {
                    RoundedRectangle(cornerRadius: 10)
                        .fill(device.online ? Color.guardianGreen.opacity(0.12) : Color.adaptiveGray5)
                        .frame(width: 44, height: 44)

                    Image(systemName: vendorIcon(device.vendor))
                        .font(.system(size: 20))
                        .foregroundColor(device.online ? .guardianGreen : .secondary)
                }

                // Device Basic Info
                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(device.online ? Color.guardianGreen : Color.secondary.opacity(0.5))
                            .frame(width: 6, height: 6)

                        Text(device.displayName)
                            .font(.subheadline.bold())
                            .foregroundColor(.primary)
                            .lineLimit(1)

                        if device.isLocked == true {
                            Text(i18n.t("locked"))
                                .font(.system(size: 10, weight: .bold))
                                .padding(.horizontal, 5)
                                .padding(.vertical, 1)
                                .background(Color.red.opacity(0.15))
                                .foregroundColor(.red)
                                .cornerRadius(4)
                        }
                    }

                    Text("IP: \(device.ip) · MAC: \(device.mac)")
                        .font(.caption2.monospaced())
                        .foregroundColor(.secondary)

                    HStack(spacing: 6) {
                        Text(device.vendor.isEmpty ? "Generic" : device.vendor)
                            .font(.system(size: 11))
                            .foregroundColor(.secondary)

                        if device.online {
                            Text("·")
                                .foregroundColor(.secondary)
                            Text("\(i18n.t("realtimeSpeed")): \(formatSpeed(device.rxRate))")
                                .font(.system(size: 11, weight: .medium, design: .monospaced))
                                .foregroundColor(.guardianGreen)
                        }
                    }
                }

                Spacer()
            }

            Divider()

            // Bottom Assignment and Actions Bar
            HStack {
                // Member Assignment Badge
                if let member = assignedMember {
                    HStack(spacing: 4) {
                        Text(avatarEmoji(member.avatar))
                            .font(.caption)
                        Text(member.name)
                            .font(.caption.bold())
                            .foregroundColor(.guardianGreen)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(Color.guardianGreen.opacity(0.12))
                    .cornerRadius(6)
                } else {
                    Text(i18n.t("unassigned"))
                        .font(.caption)
                        .foregroundColor(.secondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(Color.adaptiveGray5)
                        .cornerRadius(6)
                }

                Spacer()

                // Member Assignment Button
                Button {
                    onAssign()
                    HapticManager.impact(.light)
                } label: {
                    Text(i18n.t("btnAssign"))
                        .font(.caption.bold())
                        .foregroundColor(.guardianGreen)
                }
                .buttonStyle(.borderless)

                Text("|")
                    .foregroundColor(.secondary.opacity(0.4))
                    .font(.caption2)

                // Lock / Unlock Button
                Button {
                    let currentlyLocked = device.isLocked == true
                    HapticManager.impact(.medium)
                    Task {
                        if currentlyLocked {
                            await appState.unlockDevice(mac: device.mac)
                        } else {
                            await appState.lockDevice(mac: device.mac)
                        }
                    }
                } label: {
                    HStack(spacing: 3) {
                        Image(systemName: device.isLocked == true ? "lock.open.fill" : "lock.fill")
                            .font(.caption2)
                        Text(device.isLocked == true ? i18n.t("btnUnlock") : i18n.t("btnLock"))
                            .font(.caption.bold())
                    }
                    .foregroundColor(device.isLocked == true ? .guardianGreen : .red)
                }
                .buttonStyle(.borderless)
            }
        }
        .padding(.vertical, 4)
    }

    private var assignedMember: Member? {
        appState.members.first(where: { m in
            m.id == device.memberId || m.deviceMACs.contains(device.mac)
        })
    }

    private func formatSpeed(_ bytesPerSec: UInt64) -> String {
        if bytesPerSec >= 1024 * 1024 {
            let mb = Double(bytesPerSec) / (1024.0 * 1024.0)
            return String(format: "%.1f MB/s", mb)
        } else {
            let kb = Double(bytesPerSec) / 1024.0
            return String(format: "%.0f KB/s", kb)
        }
    }

    private func avatarEmoji(_ avatar: String) -> String {
        switch avatar {
        case "girl": return "👧"
        case "student": return "🧑‍🎓"
        case "child": return "👶"
        default: return "👦"
        }
    }

    private func vendorIcon(_ vendor: String) -> String {
        let v = vendor.lowercased()
        if v.contains("apple") {
            return "apple.logo"
        } else if v.contains("sony") || v.contains("playstation") {
            return "gamecontroller.fill"
        } else if v.contains("nintendo") {
            return "gamecontroller"
        } else if v.contains("huawei") || v.contains("xiaomi") || v.contains("samsung") || v.contains("oppo") || v.contains("vivo") {
            return "smartphone"
        } else if v.contains("intel") || v.contains("lenovo") || v.contains("dell") || v.contains("hp") {
            return "laptopcomputer"
        } else {
            return "network"
        }
    }
}
