// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "ParentControlApp",
    platforms: [
        .iOS(.v16),
        .macOS(.v13)
    ],
    products: [
        .executable(
            name: "ParentControlApp",
            targets: ["ParentControlApp"]
        )
    ],
    dependencies: [
        .package(path: "../ParentControlCore")
    ],
    targets: [
        .executableTarget(
            name: "ParentControlApp",
            dependencies: [
                .product(name: "ParentControlCore", package: "ParentControlCore")
            ],
            path: "Sources"
        )
    ]
)
