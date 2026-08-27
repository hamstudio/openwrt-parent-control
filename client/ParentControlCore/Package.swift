// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "ParentControlCore",
    platforms: [
        .iOS(.v16),
        .macOS(.v13)
    ],
    products: [
        .library(
            name: "ParentControlCore",
            targets: ["ParentControlCore"]
        ),
        .library(
            name: "ParentControlBridge",
            type: .dynamic,
            targets: ["ParentControlBridge"]
        )
    ],
    targets: [
        .target(
            name: "ParentControlCore",
            dependencies: [],
            path: "Sources/ParentControlCore"
        ),
        .target(
            name: "ParentControlBridge",
            dependencies: ["ParentControlCore"],
            path: "Sources/ParentControlBridge"
        ),
        .testTarget(
            name: "ParentControlCoreTests",
            dependencies: ["ParentControlCore"],
            path: "Tests/ParentControlCoreTests"
        )
    ]
)
