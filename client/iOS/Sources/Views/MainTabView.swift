import SwiftUI
import ParentControlCore

struct MainTabView: View {
    @EnvironmentObject var appState: AppState
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Label("家庭管控", systemImage: "person.3.sequence.fill")
                }
                .tag(0)

            DevicesListView()
                .tabItem {
                    Label("设备列表", systemImage: "laptopcomputer.and.iphone")
                }
                .tag(1)

            SettingsView()
                .tabItem {
                    Label("安全设置", systemImage: "shield.checkered")
                }
                .tag(2)
        }
        .tint(.guardianGreen)
    }
}
