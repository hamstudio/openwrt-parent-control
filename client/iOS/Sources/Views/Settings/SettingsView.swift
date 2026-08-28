import SwiftUI
import ParentControlCore

struct SettingsView: View {
    @EnvironmentObject var appState: AppState
    @ObservedObject var i18n = I18n.shared

    @State private var serverAddress: String = ""
    @State private var pinInput: String = ""
    @State private var selectedLanguage: String = "auto"
    @State private var globalEnabled: Bool = true
    @State private var safeSearch: Bool = true
    @State private var blockDoH: Bool = true
    @State private var isolateNew: Bool = false
    @State private var isDiscovering: Bool = false
    @State private var isSaving: Bool = false
    @State private var candidates: [String] = []

    var body: some View {
        NavigationStack {
            Form {
                // Language & Region
                Section(i18n.t("languageSection")) {
                    Picker(i18n.t("languageSection"), selection: $selectedLanguage) {
                        ForEach(i18n.supportedLanguages) { lang in
                            Text("\(lang.nativeName) (\(lang.name))").tag(lang.id)
                        }
                    }
                    .onChange(of: selectedLanguage) { newLang in
                        appState.appLanguage = newLang
                        HapticManager.impact(.light)
                    }
                }

                // Router Connection Configuration
                Section(i18n.t("routerConnSection")) {
                    HStack {
                        #if os(iOS)
                        TextField("http://192.168.1.1:8088", text: $serverAddress)
                            .keyboardType(.URL)
                            .textInputAutocapitalization(.never)
                            .disableAutocorrection(true)
                        #else
                        TextField("http://192.168.1.1:8088", text: $serverAddress)
                            .disableAutocorrection(true)
                        #endif

                        Button(i18n.t("btnConnect")) {
                            appState.serverURL = serverAddress
                            HapticManager.notification(.success)
                            Task {
                                await appState.refreshAll()
                            }
                        }
                        .bold()
                        .foregroundColor(.guardianGreen)
                    }

                    // PIN Code Configuration
                    HStack {
                        SecureField(i18n.t("pinSettingDesc"), text: $pinInput)
                            .keyboardType(.numberPad)

                        Button(i18n.t("btnSavePin")) {
                            appState.pinCode = pinInput
                            HapticManager.notification(.success)
                            Task {
                                await appState.refreshAll()
                            }
                        }
                        .bold()
                        .foregroundColor(.guardianGreen)
                    }

                    // Connection Status Indicator
                    HStack {
                        Circle()
                            .fill(appState.isConnected ? Color.guardianGreen : Color.red)
                            .frame(width: 8, height: 8)
                        Text(appState.isConnected ? (appState.needsPinAuth ? i18n.t("pinRequiredTitle") : i18n.t("connectedTo")) : (appState.errorMessage ?? i18n.t("notConnected")))
                            .font(.caption)
                            .foregroundColor(appState.isConnected && !appState.needsPinAuth ? .secondary : .red)
                    }

                    // Auto-Discovery
                    Button {
                        isDiscovering = true
                        HapticManager.impact(.medium)
                        Task {
                            await appState.autoDiscover()
                            serverAddress = appState.serverURL
                            isDiscovering = false
                        }
                    } label: {
                        HStack {
                            Image(systemName: "antenna.radiowaves.left.and.right")
                            Text(isDiscovering ? i18n.t("discovering") : i18n.t("btnAutoDiscover"))
                            if isDiscovering {
                                Spacer()
                                ProgressView()
                            }
                        }
                        .foregroundColor(.guardianGreen)
                    }

                    // Quick Gateway Candidates
                    if !candidates.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            Text(i18n.t("quickGateways"))
                                .font(.caption2)
                                .foregroundColor(.secondary)
                            ScrollView(.horizontal, showsIndicators: false) {
                                HStack(spacing: 8) {
                                    ForEach(candidates, id: \.self) { url in
                                        Button(url.replacingOccurrences(of: "http://", with: "").replacingOccurrences(of: ":8088", with: "")) {
                                            serverAddress = url
                                            appState.serverURL = url
                                            Task {
                                                await appState.refreshAll()
                                            }
                                        }
                                        .font(.caption.monospaced())
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 4)
                                        .background(serverAddress == url ? Color.guardianGreen.opacity(0.2) : Color.adaptiveGray5)
                                        .foregroundColor(serverAddress == url ? .guardianGreen : .primary)
                                        .cornerRadius(6)
                                    }
                                }
                            }
                        }
                        .padding(.vertical, 2)
                    }
                }

                // Global Security Policy
                Section(i18n.t("globalPolicySection")) {
                    Toggle(i18n.t("globalSwitch"), isOn: $globalEnabled)
                        .tint(.guardianGreen)
                    Toggle(i18n.t("safeSearch"), isOn: $safeSearch)
                        .tint(.guardianGreen)
                    Toggle(i18n.t("blockDoH"), isOn: $blockDoH)
                        .tint(.guardianGreen)
                    Toggle(i18n.t("isolateNew"), isOn: $isolateNew)
                        .tint(.guardianGreen)

                    Button(i18n.t("btnApplySettings")) {
                        saveSettings()
                    }
                    .bold()
                    .foregroundColor(.guardianGreen)
                    .frame(maxWidth: .infinity)
                }

                // System & Runtime Status
                Section(i18n.t("systemInfoSection")) {
                    LabeledContent(i18n.t("clientVersion"), value: "1.0.0 (Native)")
                    LabeledContent(i18n.t("dpiEngine"), value: appState.status?.kernelDpiReady == true ? "kmod-oaf (Active)" : "Inactive")
                    LabeledContent(i18n.t("uptime"), value: "\(appState.status?.uptimeSeconds ?? 0) s")
                    LabeledContent(i18n.t("detectedDevicesCount"), value: "\(appState.status?.totalDevices ?? 0)")
                }
            }
            .navigationTitle(i18n.t("settingsTitle"))
            .onAppear {
                serverAddress = appState.serverURL
                pinInput = appState.pinCode
                selectedLanguage = appState.appLanguage
                globalEnabled = appState.settings.enabled
                safeSearch = appState.settings.enforceSafeSearch
                blockDoH = appState.settings.blockDoHDoT
                isolateNew = appState.settings.isolateNewDevices
                Task {
                    candidates = await RouterDiscovery.shared.getCandidateURLs()
                }
            }
        }
    }

    private func saveSettings() {
        isSaving = true
        let newSettings = GlobalSettings(
            enabled: globalEnabled,
            enforceSafeSearch: safeSearch,
            blockDoHDoT: blockDoH,
            isolateNewDevices: isolateNew
        )

        Task {
            let success = await appState.saveSettings(newSettings)
            if success {
                HapticManager.notification(.success)
            }
            isSaving = false
        }
    }
}
