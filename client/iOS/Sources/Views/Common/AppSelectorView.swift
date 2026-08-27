import SwiftUI
import ParentControlCore

struct AppSelectorView: View {
    @Binding var selectedAppIDs: [Int]
    let categories: [AppCategory]

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text("深度应用封禁 (L7 DPI)")
                    .font(.subheadline.bold())
                Spacer()
                Text("已选 \(selectedAppIDs.count) 款 App")
                    .font(.caption)
                    .foregroundColor(.guardianGreen)
            }

            if categories.isEmpty {
                Text("加载特征库中...")
                    .font(.caption)
                    .foregroundColor(.secondary)
            } else {
                ForEach(categories, id: \.classId) { category in
                    CategoryAppGroupView(
                        category: category,
                        selectedAppIDs: $selectedAppIDs
                    )
                }
            }
        }
    }
}

struct CategoryAppGroupView: View {
    let category: AppCategory
    @Binding var selectedAppIDs: [Int]
    @State private var isExpanded: Bool = true

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Button {
                    withAnimation { isExpanded.toggle() }
                } label: {
                    HStack(spacing: 6) {
                        Image(systemName: isExpanded ? "chevron.down" : "chevron.right")
                            .font(.caption2.bold())
                            .foregroundColor(.secondary)

                        Text(category.classZh)
                            .font(.subheadline.bold())
                            .foregroundColor(.primary)
                    }
                }

                Spacer()

                Button {
                    toggleSelectAll()
                } label: {
                    Text(isAllSelected ? "取消全选" : "全选")
                        .font(.caption2)
                        .foregroundColor(.guardianGreen)
                }
            }

            if isExpanded {
                FlowLayout(spacing: 6) {
                    ForEach(category.apps, id: \.id) { app in
                        let isSelected = selectedAppIDs.contains(app.id)
                        Button {
                            toggleApp(app.id)
                            HapticManager.impact(.light)
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                                    .font(.caption2)
                                Text(app.name)
                                    .font(.caption)
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(isSelected ? Color.guardianGreen.opacity(0.15) : Color.adaptiveSecondaryBackground)
                            .foregroundColor(isSelected ? .guardianGreen : .primary)
                            .cornerRadius(8)
                            .overlay(
                                RoundedRectangle(cornerRadius: 8)
                                    .stroke(isSelected ? Color.guardianGreen : Color.clear, lineWidth: 1)
                            )
                        }
                    }
                }
            }
        }
        .padding(10)
        .background(Color.adaptiveSecondaryBackground.opacity(0.5))
        .cornerRadius(12)
    }

    private var isAllSelected: Bool {
        let appIDs = Set(category.apps.map { $0.id })
        let currentSelected = Set(selectedAppIDs)
        return appIDs.isSubset(of: currentSelected)
    }

    private func toggleSelectAll() {
        var set = Set(selectedAppIDs)
        let appIDs = category.apps.map { $0.id }
        if isAllSelected {
            for id in appIDs { set.remove(id) }
        } else {
            for id in appIDs { set.insert(id) }
        }
        selectedAppIDs = Array(set)
    }

    private func toggleApp(_ id: Int) {
        if let idx = selectedAppIDs.firstIndex(of: id) {
            selectedAppIDs.remove(at: idx)
        } else {
            selectedAppIDs.append(id)
        }
    }
}

// MARK: - FlowLayout 标签流式排版
struct FlowLayout: Layout {
    var spacing: CGFloat = 6

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? 0
        var height: CGFloat = 0
        var currentX: CGFloat = 0
        var currentY: CGFloat = 0
        var maxHeightInRow: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if currentX + size.width > width && currentX > 0 {
                currentX = 0
                currentY += maxHeightInRow + spacing
                maxHeightInRow = 0
            }
            maxHeightInRow = max(maxHeightInRow, size.height)
            currentX += size.width + spacing
        }
        height = currentY + maxHeightInRow
        return CGSize(width: width, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var currentX = bounds.minX
        var currentY = bounds.minY
        var maxHeightInRow: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if currentX + size.width > bounds.maxX && currentX > bounds.minX {
                currentX = bounds.minX
                currentY += maxHeightInRow + spacing
                maxHeightInRow = 0
            }
            subview.place(at: CGPoint(x: currentX, y: currentY), proposal: .unspecified)
            maxHeightInRow = max(maxHeightInRow, size.height)
            currentX += size.width + spacing
        }
    }
}
