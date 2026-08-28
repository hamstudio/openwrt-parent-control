import SwiftUI
import ParentControlCore

struct DeviceAssignSheet: View {
    @Environment(\.dismiss) var dismiss
    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared

    let device: Device
    @State private var selectedMemberId: String = ""
    @State private var isSubmitting: Bool = false

    init(device: Device) {
        self.device = device
        _selectedMemberId = State(initialValue: device.memberId ?? "")
    }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(device.displayName)
                            .font(.headline)
                        Text("IP: \(device.ip) · MAC: \(device.mac)")
                            .font(.caption.monospaced())
                            .foregroundColor(.secondary)
                    }
                    .padding(.vertical, 4)
                }

                Section(i18n.t("assignModalSubtitle")) {
                    // Unassigned option
                    Button {
                        selectedMemberId = ""
                        HapticManager.impact(.light)
                    } label: {
                        HStack {
                            ZStack {
                                RoundedRectangle(cornerRadius: 8)
                                .fill(Color.adaptiveGray5)
                                .frame(width: 36, height: 36)
                                Text("🚫")
                                    .font(.subheadline)
                            }

                            VStack(alignment: .leading, spacing: 2) {
                                Text(i18n.t("unbindDevice"))
                                    .font(.subheadline.bold())
                                    .foregroundColor(.primary)
                                Text(i18n.t("unassigned"))
                                    .font(.caption2)
                                    .foregroundColor(.secondary)
                            }

                            Spacer()

                            if selectedMemberId.isEmpty {
                                Image(systemName: "checkmark")
                                    .foregroundColor(.guardianGreen)
                                    .bold()
                            }
                        }
                    }

                    // Family member list
                    ForEach(appState.members) { member in
                        Button {
                            selectedMemberId = member.id
                            HapticManager.impact(.light)
                        } label: {
                            HStack {
                                ZStack {
                                    RoundedRectangle(cornerRadius: 8)
                                        .fill(Color.guardianGreen.opacity(0.12))
                                        .frame(width: 36, height: 36)
                                    Text(avatarEmoji(member.avatar))
                                        .font(.title3)
                                }

                                VStack(alignment: .leading, spacing: 2) {
                                    Text(member.name)
                                        .font(.subheadline.bold())
                                        .foregroundColor(.primary)
                                    Text("\(i18n.t("statDevices")): \(member.deviceMACs.count) · \(i18n.t("todayUsage")): \(member.usedMinutes) \(i18n.t("minutes"))")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }

                                Spacer()

                                if selectedMemberId == member.id {
                                    Image(systemName: "checkmark")
                                        .foregroundColor(.guardianGreen)
                                        .bold()
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle(i18n.t("assignModalTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(i18n.t("cancel")) {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(i18n.t("btnSave")) {
                        saveAssignment()
                    }
                    .bold()
                    .foregroundColor(.guardianGreen)
                    .disabled(isSubmitting)
                }
            }
        }
    }

    private func saveAssignment() {
        isSubmitting = true
        Task {
            let success = await appState.assignDevice(mac: device.mac, memberId: selectedMemberId.isEmpty ? nil : selectedMemberId)
            if success {
                HapticManager.notification(.success)
                dismiss()
            }
            isSubmitting = false
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
}
