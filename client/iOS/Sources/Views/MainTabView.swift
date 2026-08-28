import SwiftUI
import ParentControlCore

struct MainTabView: View {
    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared
    @State private var selectedTab = 0

    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Label(i18n.t("tabMembers"), systemImage: "person.3.sequence.fill")
                }
                .tag(0)

            DevicesListView()
                .tabItem {
                    Label(i18n.t("tabDevices"), systemImage: "laptopcomputer.and.iphone")
                }
                .tag(1)

            SettingsView()
                .tabItem {
                    Label(i18n.t("tabSettings"), systemImage: "shield.checkered")
                }
                .tag(2)
        }
        .tint(.guardianGreen)
    }
}
