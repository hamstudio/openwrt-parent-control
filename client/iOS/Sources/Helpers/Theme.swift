import SwiftUI

public extension Color {
    static var adaptiveBackground: Color {
        #if os(iOS)
        return Color(uiColor: .systemBackground)
        #elseif os(macOS)
        return Color(nsColor: .windowBackgroundColor)
        #else
        return Color.white
        #endif
    }

    static var adaptiveSecondaryBackground: Color {
        #if os(iOS)
        return Color(uiColor: .secondarySystemBackground)
        #elseif os(macOS)
        return Color(nsColor: .controlBackgroundColor)
        #else
        return Color.gray.opacity(0.1)
        #endif
    }

    static var adaptiveGroupedBackground: Color {
        #if os(iOS)
        return Color(uiColor: .systemGroupedBackground)
        #elseif os(macOS)
        return Color(nsColor: .underPageBackgroundColor)
        #else
        return Color.gray.opacity(0.15)
        #endif
    }

    static var adaptiveGray5: Color {
        #if os(iOS)
        return Color(uiColor: .systemGray5)
        #elseif os(macOS)
        return Color(nsColor: .separatorColor)
        #else
        return Color.gray.opacity(0.2)
        #endif
    }

    // MARK: - Health Guardian Green Brand Colors
    static var guardianGreen: Color {
        return Color(red: 16.0 / 255.0, green: 185.0 / 255.0, blue: 129.0 / 255.0)
    }

    static var emerald: Color {
        return Color(red: 5.0 / 255.0, green: 150.0 / 255.0, blue: 105.0 / 255.0)
    }

    static var brand: Color {
        return guardianGreen
    }
}
