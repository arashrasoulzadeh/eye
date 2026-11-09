// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "EyeSDK",
    platforms: [.iOS(.v13), .macOS(.v11)],
    products: [
        .library(name: "EyeSDK", targets: ["EyeSDK"])
    ],
    targets: [
        .target(name: "EyeSDK"),
        .testTarget(name: "EyeSDKTests", dependencies: ["EyeSDK"]),
    ]
)
