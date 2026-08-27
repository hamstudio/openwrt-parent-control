import SwiftUI
import ParentControlCore

struct MemberCardView: View {
    let member: Member
    @EnvironmentObject var appState: AppState
    @State private var showingEditSheet = false

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            // 头部信息
            HStack(spacing: 12) {
                Text(avatarEmoji(for: member.avatar))
                    .font(.system(size: 32))
                    .frame(width: 48, height: 48)
                    .background(Color.guardianGreen.opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))

                VStack(alignment: .leading, spacing: 3) {
                    HStack(spacing: 8) {
                        Text(member.name)
                            .font(.headline)
                            .fontWeight(.bold)

                        statusBadge
                    }

                    Text("绑定 \(member.deviceMACs.count) 台设备 · 限制 \(member.blockedAppIDs.count) 款 App")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                Spacer()

                Button {
                    showingEditSheet = true
                    HapticManager.impact(.light)
                } label: {
                    Image(systemName: "slider.horizontal.3")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(.guardianGreen)
                        .padding(8)
                        .background(Color.guardianGreen.opacity(0.1))
                        .clipShape(Circle())
                }
            }

            // 多时间段管控摘要
            if member.schedule.enabled && !member.schedule.timeRanges.isEmpty {
                HStack(spacing: 6) {
                    Image(systemName: "clock.badge.checkmark")
                        .font(.caption2)
                        .foregroundColor(.guardianGreen)

                    Text(scheduleSummaryText)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color.guardianGreen.opacity(0.08))
                .cornerRadius(8)
            }

            // 时长配额进度
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    Text("今日活跃上网时长")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Spacer()
                    Text("\(member.usedMinutes) / \(member.quotaMinutes > 0 ? "\(member.quotaMinutes) 分钟" : "不限时")")
                        .font(.caption.bold())
                        .foregroundColor(.primary)
                }

                GeometryReader { geo in
                    ZStack(alignment: .leading) {
                        Capsule()
                            .fill(Color.adaptiveGray5)
                            .frame(height: 8)

                        Capsule()
                            .fill(progressColor)
                            .frame(width: max(0, min(geo.size.width, geo.size.width * CGFloat(member.quotaProgress))), height: 8)
                            .animation(.spring(), value: member.quotaProgress)
                    }
                }
                .frame(height: 8)
            }
            .padding(10)
            .background(Color.adaptiveSecondaryBackground)
            .cornerRadius(12)

            // 操作按钮组
            HStack(spacing: 10) {
                if member.isLocked {
                    Button {
                        HapticManager.impact(.medium)
                        Task { await appState.unlockMember(id: member.id) }
                    } label: {
                        Label("恢复上网", systemImage: "lock.open.fill")
                            .font(.subheadline.bold())
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(Color.guardianGreen)
                            .foregroundColor(.white)
                            .cornerRadius(12)
                    }
                } else {
                    Button {
                        HapticManager.impact(.heavy)
                        Task { await appState.lockMember(id: member.id) }
                    } label: {
                        Label("一键断网", systemImage: "lock.fill")
                            .font(.subheadline.bold())
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(Color.red)
                            .foregroundColor(.white)
                            .cornerRadius(12)
                    }
                }

                Button {
                    HapticManager.impact(.medium)
                    Task { await appState.bonusMember(id: member.id, minutes: 30) }
                } label: {
                    Label("奖励 +30分", systemImage: "plus.circle.fill")
                        .font(.subheadline.bold())
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(Color.orange)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                }
            }
        }
        .padding(16)
        .background(Color.adaptiveBackground)
        .cornerRadius(18)
        .shadow(color: Color.black.opacity(0.04), radius: 8, x: 0, y: 3)
        .sheet(isPresented: $showingEditSheet) {
            MemberEditorView(member: member)
        }
    }

    private var scheduleSummaryText: String {
        let isBlock = (member.schedule.action == "block")
        let actionStr = isBlock ? "🚫 禁网" : "✅ 允许"
        let rangesStr = member.schedule.timeRanges.map { "\($0.startTime)~\($0.endTime)" }.joined(separator: ", ")
        return "\(actionStr): \(rangesStr)"
    }

    private var statusBadge: some View {
        Group {
            if member.isLocked {
                Text("已断网")
                    .font(.caption2.bold())
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.red.opacity(0.15))
                    .foregroundColor(.red)
                    .cornerRadius(6)
            } else if member.isBonusActive {
                Text("加时中")
                    .font(.caption2.bold())
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.orange.opacity(0.15))
                    .foregroundColor(.orange)
                    .cornerRadius(6)
            } else if member.isQuotaExceeded {
                Text("限额耗尽")
                    .font(.caption2.bold())
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.orange.opacity(0.15))
                    .foregroundColor(.orange)
                    .cornerRadius(6)
            } else {
                Text("正常")
                    .font(.caption2.bold())
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(Color.guardianGreen.opacity(0.15))
                    .foregroundColor(.guardianGreen)
                    .cornerRadius(6)
            }
        }
    }

    private var progressColor: Color {
        if member.quotaProgress > 0.9 {
            return .red
        } else if member.quotaProgress > 0.7 {
            return .orange
        } else {
            return .guardianGreen
        }
    }

    private func avatarEmoji(for avatar: String) -> String {
        switch avatar {
        case "boy": return "👦"
        case "girl": return "👧"
        case "student": return "🧑‍🎓"
        case "child": return "👶"
        default: return "👦"
        }
    }
}
