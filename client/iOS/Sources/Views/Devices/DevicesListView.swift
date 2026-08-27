import SwiftUI
import ParentControlCore

struct DevicesListView: View {
    @EnvironmentObject var appState: AppState
    @State private var searchText: String = ""
    @State private var onlyOnline: Bool = false

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Toggle("仅显示当前在线设备", isOn: $onlyOnline)
                        .font(.subheadline)
                }

                Section("已发现设备 (\(filteredDevices.count) 台)") {
                    if filteredDevices.isEmpty {
                        Text("未找到匹配设备")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    } else {
                        ForEach(filteredDevices) { device in
                            DeviceRowView(device: device)
                        }
                    }
                }
            }
            .navigationTitle("局域网设备")
            .searchable(text: $searchText, prompt: "搜索设备名、IP 或 MAC")
            .refreshable {
                await appState.refreshAll()
            }
        }
    }

    private var filteredDevices: [Device] {
        appState.devices.filter { d in
            let matchesSearch = searchText.isEmpty ||
                d.hostname.localizedCaseInsensitiveContains(searchText) ||
                d.ip.contains(searchText) ||
                d.mac.localizedCaseInsensitiveContains(searchText) ||
                d.vendor.localizedCaseInsensitiveContains(searchText)

            let matchesOnline = !onlyOnline || d.online
            return matchesSearch && matchesOnline
        }
    }
}

struct DeviceRowView: View {
    let device: Device
    @EnvironmentObject var appState: AppState

    var body: some View {
        HStack(spacing: 12) {
            ZStack {
                Circle()
                    .fill(device.online ? Color.green.opacity(0.15) : Color.adaptiveGray5)
                    .frame(width: 40, height: 40)

                Image(systemName: vendorIcon(device.vendor))
                    .foregroundColor(device.online ? .green : .secondary)
            }

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    Text(device.displayName)
                        .font(.subheadline.bold())

                    if let memberName = assignedMemberName {
                        Text(memberName)
                            .font(.caption2.bold())
                            .padding(.horizontal, 6)
                            .padding(.vertical, 1)
                            .background(Color.guardianGreen.opacity(0.15))
                            .foregroundColor(.guardianGreen)
                            .cornerRadius(4)
                    }
                }

                Text("\(device.ip) · \(device.mac)")
                    .font(.caption2.monospaced())
                    .foregroundColor(.secondary)

                Text(device.vendor)
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }

            Spacer()

            if device.online {
                VStack(alignment: .trailing, spacing: 2) {
                    Text(device.formattedSpeed)
                        .font(.caption.bold().monospaced())
                        .foregroundColor(.guardianGreen)
                    Text("实时速率")
                        .font(.system(size: 9))
                        .foregroundColor(.secondary)
                }
            } else {
                Text("离线")
                    .font(.caption2)
                    .foregroundColor(.secondary)
            }
        }
        .padding(.vertical, 4)
    }

    private var assignedMemberName: String? {
        appState.members.first(where: { $0.deviceMACs.contains(device.mac) })?.name
    }

    private func vendorIcon(_ vendor: String) -> String {
        switch vendor.lowercased() {
        case let v where v.contains("apple"):
            return "apple.logo"
        case let v where v.contains("sony"):
            return "gamecontroller.fill"
        case let v where v.contains("nintendo"):
            return "gamecontroller"
        case let v where v.contains("huawei") || v.contains("xiaomi"):
            return "smartphone"
        default:
            return "network"
        }
    }
}
