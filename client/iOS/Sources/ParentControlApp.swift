import SwiftUI
import ParentControlCore

@main
struct ParentControlApp: App {
    @StateObject private var appState = AppState.shared

    var body: some Scene {
        WindowGroup {
            MainTabView()
                .environmentObject(appState)
                .onAppear {
                    appState.startAutoRefresh()
                }
        }
    }
}
