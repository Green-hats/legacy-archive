import Foundation

@MainActor
final class DictationViewModel: ObservableObject {
    @Published var mode: DictationMode = .paper {
        didSet {
            guard oldValue != mode else { return }
            stop(showMessage: false)
        }
    }

    @Published var status: PlaybackStatus = .stopped
    @Published var words: [String] = []
    @Published var currentIndex = 0
    @Published var mistakeCount = 0
    @Published var answer = ""
    @Published var feedback = "请先导入词单。"
    @Published var settings = DictationSettings()
    @Published var message: String?

    private let speech = SpeechService()
    private var playbackTask: Task<Void, Never>?

    var canStart: Bool {
        !words.isEmpty && status == .stopped
    }

    var canControlPlayback: Bool {
        !words.isEmpty && status != .stopped
    }

    var progressText: String {
        guard !words.isEmpty else { return "等待开始" }
        let visibleIndex = min(currentIndex + 1, words.count)
        return "当前进度：\(visibleIndex)/\(words.count)"
    }

    var progressFraction: Double {
        guard !words.isEmpty else { return 0 }
        return Double(min(currentIndex, words.count)) / Double(words.count)
    }

    func importWords(from url: URL) {
        let didStartAccessing = url.startAccessingSecurityScopedResource()
        defer {
            if didStartAccessing {
                url.stopAccessingSecurityScopedResource()
            }
        }

        do {
            words = try WordListParser.parse(url: url)
            currentIndex = 0
            mistakeCount = 0
            answer = ""
            feedback = "已导入 \(words.count) 个单词。"
            message = feedback
        } catch {
            message = "导入失败：\(error.localizedDescription)"
        }
    }

    func start() {
        guard canStart else { return }

        playbackTask?.cancel()
        speech.cancel()

        currentIndex = 0
        mistakeCount = 0
        answer = ""
        status = .playing

        switch mode {
        case .paper:
            feedback = "纸质听写播放中。"
            playbackTask = Task { [weak self] in
                await self?.runPaperMode()
            }
        case .interactive:
            feedback = "请听单词并输入拼写。"
            playbackTask = Task { [weak self] in
                await self?.speakCurrentWord()
            }
        }
    }

    func togglePause() {
        switch status {
        case .playing:
            status = .paused
            speech.pause()
        case .paused:
            status = .playing
            speech.resume()
        case .stopped:
            break
        }
    }

    func stop(showMessage: Bool = true) {
        playbackTask?.cancel()
        playbackTask = nil
        speech.cancel()
        status = .stopped
        currentIndex = 0
        answer = ""
        feedback = words.isEmpty ? "请先导入词单。" : "已停止。"
        if showMessage {
            message = "听写已结束。"
        }
    }

    func shuffleWords() {
        guard status == .stopped, !words.isEmpty else { return }
        words.shuffle()
        currentIndex = 0
        answer = ""
        feedback = "已打乱单词顺序。"
        message = feedback
    }

    func replayCurrentWord() {
        guard status != .stopped, currentIndex < words.count else { return }
        playbackTask?.cancel()
        playbackTask = Task { [weak self] in
            await self?.speakCurrentWord()
        }
    }

    func submitAnswer() {
        guard mode == .interactive, status == .playing, currentIndex < words.count else { return }

        let correct = words[currentIndex]
        let normalizedAnswer = normalize(answer)

        guard !normalizedAnswer.isEmpty else {
            message = "请输入拼写后再提交。"
            return
        }

        if normalizedAnswer == normalize(correct) {
            feedback = "正确。"
            answer = ""
            currentIndex += 1

            if currentIndex >= words.count {
                finishInteractive()
            } else {
                playbackTask?.cancel()
                playbackTask = Task { [weak self] in
                    try? await Task.sleep(nanoseconds: 600_000_000)
                    await self?.speakCurrentWord()
                }
            }
        } else {
            mistakeCount += 1
            feedback = "错误，正确答案：\(correct)"
            answer = ""
        }
    }

    func skipWord() {
        guard mode == .interactive, status == .playing, currentIndex < words.count else { return }

        let skipped = words[currentIndex]
        mistakeCount += 1
        currentIndex += 1
        answer = ""
        feedback = "已跳过：\(skipped)"

        if currentIndex >= words.count {
            finishInteractive()
        } else {
            playbackTask?.cancel()
            playbackTask = Task { [weak self] in
                try? await Task.sleep(nanoseconds: 600_000_000)
                await self?.speakCurrentWord()
            }
        }
    }

    private func runPaperMode() async {
        while currentIndex < words.count && !Task.isCancelled {
            await waitWhilePaused()
            guard status == .playing, currentIndex < words.count else { break }

            let word = words[currentIndex]
            await speak(word)
            guard status == .playing, !Task.isCancelled else { break }

            currentIndex += 1

            if currentIndex < words.count {
                await delay(seconds: settings.gapSeconds)
            }
        }

        guard !Task.isCancelled, status == .playing, currentIndex >= words.count else { return }
        status = .stopped
        feedback = "纸质听写完成，答案已显示。"
        message = "纸质听写完成。"
    }

    private func speakCurrentWord() async {
        guard currentIndex < words.count, status != .stopped else { return }
        await waitWhilePaused()
        guard status != .stopped, currentIndex < words.count else { return }
        await speak(words[currentIndex])
    }

    private func speak(_ word: String) async {
        let repeats = max(1, min(settings.repeatCount, 3))

        for repeatIndex in 0..<repeats {
            guard status != .stopped, !Task.isCancelled else { return }
            await waitWhilePaused()
            await speech.speak(word, rate: settings.speechRate)

            if repeatIndex < repeats - 1 {
                await delay(seconds: 0.6)
            }
        }
    }

    private func waitWhilePaused() async {
        while status == .paused && !Task.isCancelled {
            try? await Task.sleep(nanoseconds: 100_000_000)
        }
    }

    private func delay(seconds: Double) async {
        let clamped = max(0, seconds)
        try? await Task.sleep(nanoseconds: UInt64(clamped * 1_000_000_000))
    }

    private func finishInteractive() {
        playbackTask?.cancel()
        speech.cancel()
        status = .stopped
        feedback = "听写完成，错误次数：\(mistakeCount)"
        message = "在线听写完成。"
    }

    private func normalize(_ value: String) -> String {
        value
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
            .replacingOccurrences(of: "’", with: "'")
            .components(separatedBy: .whitespacesAndNewlines)
            .filter { !$0.isEmpty }
            .joined(separator: " ")
    }
}
