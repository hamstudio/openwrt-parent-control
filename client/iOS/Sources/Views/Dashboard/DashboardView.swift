import SwiftUI
import ParentControlCore

struct DashboardView: View {
    @EnvironmentObject var appState: AppState
    @State private var showingAddSheet = false

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    // 连接状态与指标概览
                    metricsHeader

                    // 成员卡片列表
                    if appState.members.isEmpty {
                        emptyStateView
                    } else {
                        LazyVStack(spacing: 14) {
                            ForEach(appState.members) { member in
                                MemberCardView(member: member)
                            }
                        }
                    }
                }
                .padding()
            }
            .background(Color.adaptiveGroupedBackground)
            .navigationTitle("绿色健康守护中心")
            .toolbar {
                ToolbarItem(placement: .automatic) {
                    Button {
                        showingAddSheet = true
                        HapticManager.impact(.light)
                    } label: {
                        Image(systemName: "person.crop.circle.badge.plus")
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundColor(.guardianGreen)
                    }
                }
            }
            .refreshable {
                await appState.refreshAll()
            }
            .sheet(isPresented: $showingAddSheet) {
                MemberEditorView(member: nil)
            }
        }
    }

    private var metricsHeader: some View {
        VStack(spacing: 12) {
            // 路由器连接状态指示条
            HStack {
                Circle()
                    .fill(appState.isConnected ? Color.guardianGreen : Color.red)
                    .frame(width: 8, height: 8)

                Text(appState.isConnected ? "已直连路由器 (\(appState.serverURL))" : "未连接至路由器")
                    .font(.caption.bold())
                    .foregroundColor(appState.isConnected ? .secondary : .red)

                Spacer()

                if let status = appState.status, status.kernelDpiReady {
                    HStack(spacing: 4) {
                        Image(systemName: "checkmark.shield.fill")
                            .foregroundColor(.guardianGreen)
                            .font(.caption2)
                        Text("DPI 就绪")
                            .font(.caption2.bold())
                            .foregroundColor(.guardianGreen)
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.guardianGreen.opacity(0.12))
                    .cornerRadius(6)
                }
            }

            // 统计卡片网格
            HStack(spacing: 12) {
                MetricTile(
                    title: "受管成员",
                    value: "\(appState.members.count)",
                    icon: "person.2.fill",
                    tint: .guardianGreen
                )
                MetricTile(
                    title: "受管设备",
                    value: "\(appState.status?.activeDevices ?? 0) / \(appState.status?.totalDevices ?? 0)",
                    icon: "laptopcomputer.and.iphone",
                    tint: .teal
                )
                MetricTile(
                    title: "App 特征库",
                    value: "\(appState.status?.appCount ?? 0) 款",
                    icon: "square.grid.2x2.fill",
                    tint: .emerald
                )
            }
        }
        .padding(14)
        .background(Color.adaptiveBackground)
        .cornerRadius(16)
        .shadow(color: Color.black.opacity(0.03), radius: 6, x: 0, y: 2)
    }

    private var emptyStateView: some View {
        VStack(spacing: 16) {
            Image(systemName: "shield.lefthalf.filled.badge.checkmark")
                .font(.system(size: 56))
                .foregroundColor(.guardianGreen.opacity(0.8))
                .padding(.top, 40)

            Text("暂无受管家庭成员")
                .font(.headline)

            Text("点击下方按钮添加孩子的手机、平板或电脑，开启多时段健康上网与 App 管控。")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)

            Button {
                showingAddSheet = true
                HapticManager.impact(.medium)
            } label: {
                Label("添加受管成员", systemImage: "plus.circle.fill")
                    .font(.headline)
                    .foregroundColor(.white)
                    .padding(.horizontal, 24)
                    .padding(.vertical, 12)
                    .background(Color.guardianGreen)
                    .cornerRadius(14)
            }
            .padding(.top, 8)
            .padding(.bottom, 40)
        }
        .frame(maxWidth: .infinity)
        .background(Color.adaptiveBackground)
        .cornerRadius(18)
    }
}

struct MetricTile: View {
    let title: String
    let value: String
    let icon: String
    let tint: Color

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(title)
                    .font(.caption2)
                    .foregroundColor(.secondary)
                Spacer()
                Image(systemName: icon)
                    .font(.caption)
                    .foregroundColor(tint)
            }

            Text(value)
                .font(.subheadline.bold())
                .foregroundColor(.primary)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.adaptiveSecondaryBackground)
        .cornerRadius(10)
    }
}
