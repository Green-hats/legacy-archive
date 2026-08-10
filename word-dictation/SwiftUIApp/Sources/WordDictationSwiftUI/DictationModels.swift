import Foundation

enum DictationMode: String, CaseIterable, Identifiable {
    case paper
    case interactive

    var id: String { rawValue }

    var title: String {
        switch self {
        case .paper:
            "纸质模式"
        case .interactive:
            "在线模式"
        }
    }
}

enum PlaybackStatus: Equatable {
    case stopped
    case playing
    case paused

    var title: String {
        switch self {
        case .stopped:
            "未播放"
        case .playing:
            "播放中"
        case .paused:
            "已暂停"
        }
    }
}

struct DictationSettings {
    var speechRate: Double = 0.45
    var gapSeconds: Double = 1.5
    var repeatCount: Int = 1
}

struct WordListParser {
    static func parse(url: URL) throws -> [String] {
        let text = try String(contentsOf: url, encoding: .utf8)
        let ext = url.pathExtension.lowercased()
        let rawWords: [String]

        if ext == "json" {
            rawWords = try parseJSONWords(from: text)
        } else {
            rawWords = text
                .components(separatedBy: CharacterSet(charactersIn: "\n\r,;，；"))
        }

        var seen = Set<String>()
        let words = rawWords.compactMap { value -> String? in
            let word = value.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !word.isEmpty else { return nil }

            let key = word.lowercased()
            guard !seen.contains(key) else { return nil }
            seen.insert(key)
            return word
        }

        guard !words.isEmpty else {
            throw WordListError.empty
        }

        return words
    }

    private static func parseJSONWords(from text: String) throws -> [String] {
        let data = Data(text.utf8)
        let value = try JSONSerialization.jsonObject(with: data)

        if let words = value as? [String] {
            return words
        }

        if let objects = value as? [[String: Any]] {
            return objects.compactMap { object in
                object["word"] as? String
                    ?? object["en"] as? String
                    ?? object["text"] as? String
            }
        }

        throw WordListError.invalidJSON
    }
}

enum WordListError: LocalizedError {
    case empty
    case invalidJSON

    var errorDescription: String? {
        switch self {
        case .empty:
            "没有找到有效单词。"
        case .invalidJSON:
            "JSON 词单必须是字符串数组，或包含 word/en/text 字段的对象数组。"
        }
    }
}
