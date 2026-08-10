// swift-tools-version: 5.9

import PackageDescription

let package = Package(
    name: "WordDictationSwiftUI",
    platforms: [
        .iOS(.v16),
        .macOS(.v13)
    ],
    products: [
        .executable(
            name: "WordDictationSwiftUI",
            targets: ["WordDictationSwiftUI"]
        )
    ],
    targets: [
        .executableTarget(
            name: "WordDictationSwiftUI"
        )
    ]
)
