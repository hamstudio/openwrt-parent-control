import SwiftUI
import ParentControlCore

struct DashboardView: View {
    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared

    @State private var showingAddSheet = false
    @State private var inputPin = ""

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 16) {
                    // Connection Status and Metric Overview
                    metricsHeader

                    // PIN Authentication Banner
                    if appState.needsPinAuth {
                        pinAuthBanner
                    }

                    // Member Card List
                    if appState.members.isEmpty && !appState.needsPinAuth {
                        emptyStateView
                    } else if !appState.members.isEmpty {
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
            .navigationTitle(i18n.t("appTitle"))
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

    private var pinAuthBanner: some View {
        VStack(spacing: 12) {
            HStack {
                Image(systemName: "lock.shield.fill")
                    .foregroundColor(.orange)
                    .font(.title2)
                VStack(alignment: .leading, spacing: 2) {
                    Text(i18n.t("pinRequiredTitle"))
                        .font(.headline)
                        .foregroundColor(.primary)
                    Text(i18n.t("pinRequiredDesc"))
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            HStack {
                SecureField(i18n.t("pinPlaceholder"), text: $inputPin)
                    .textFieldStyle(.roundedBorder)
                    .keyboardType(.numberPad)

                Button(i18n.t("btnVerify")) {
                    appState.pinCode = inputPin
                    HapticManager.impact(.medium)
                }
                .bold()
                .padding(.horizontal, 12)
                .padding(.vertical, 7)
                .background(Color.guardianGreen)
                .foregroundColor(.white)
                .cornerRadius(8)
            }
        }
        .padding(14)
        .background(Color.adaptiveBackground)
        .cornerRadius(16)
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(Color.orange.opacity(0.4), lineWidth: 1.5)
        )
    }

    private var metricsHeader: some View {
        VStack(spacing: 12) {
            // Router Connection Status Indicator
            HStack {
                Circle()
                    .fill(appState.isConnected ? Color.guardianGreen : Color.red)
                    .frame(width: 8, height: 8)

                Text(appState.isConnected ? "\(i18n.t("connectedTo")) (\(appState.serverURL))" : i18n.t("notConnected"))
                    .font(.caption.bold())
                    .foregroundColor(appState.isConnected ? .secondary : .red)

                Spacer()

                if let status = appState.status, status.kernelDpiReady {
                    HStack(spacing: 4) {
                        Image(systemName: "checkmark.shield.fill")
                            .foregroundColor(.guardianGreen)
                            .font(.caption2)
                        Text(i18n.t("dpiReady"))
                            .font(.caption2.bold())
                            .foregroundColor(.guardianGreen)
                    }
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.guardianGreen.opacity(0.12))
                    .cornerRadius(6)
                }
            }

            // Statistics Metric Tiles Grid
            HStack(spacing: 12) {
                MetricTile(
                    title: i18n.t("statMembers"),
                    value: "\(appState.members.count)",
                    icon: "person.2.fill",
                    tint: .guardianGreen
                )
                MetricTile(
                    title: i18n.t("statDevices"),
                    value: "\(appState.status?.activeDevices ?? 0) / \(appState.status?.totalDevices ?? 0)",
                    icon: "laptopcomputer.and.iphone",
                    tint: .teal
                )
                MetricTile(
                    title: i18n.t("statApps"),
                    value: "\(appState.status?.appCount ?? 0)",
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

            Text(i18n.t("emptyMembersTitle"))
                .font(.headline)

            Text(i18n.t("emptyMembersDesc"))
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)

            Button {
                showingAddSheet = true
                HapticManager.impact(.medium)
            } label: {
                Label(i18n.t("addMemberBtn"), systemImage: "plus.circle.fill")
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
