import SwiftUI

public enum HapticImpactStyle {
    case light
    case medium
    case heavy
}

public enum HapticNotificationType {
    case success
    case warning
    case error
}

public struct HapticManager {
    public static func impact(_ style: HapticImpactStyle = .medium) {
        #if os(iOS)
        let uiStyle: UIImpactFeedbackGenerator.FeedbackStyle
        switch style {
        case .light: uiStyle = .light
        case .medium: uiStyle = .medium
        case .heavy: uiStyle = .heavy
        }
        let generator = UIImpactFeedbackGenerator(style: uiStyle)
        generator.prepare()
        generator.impactOccurred()
        #endif
    }

    public static func notification(_ type: HapticNotificationType) {
        #if os(iOS)
        let uiType: UINotificationFeedbackGenerator.FeedbackType
        switch type {
        case .success: uiType = .success
        case .warning: uiType = .warning
        case .error: uiType = .error
        }
        let generator = UINotificationFeedbackGenerator()
        generator.prepare()
        generator.notificationOccurred(uiType)
        #endif
    }
}
