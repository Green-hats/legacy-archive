import SwiftUI
import UniformTypeIdentifiers
#if os(macOS)
import AppKit
#endif

struct ContentView: View {
    @StateObject private var viewModel = DictationViewModel()
    @State private var isImporterPresented = false
    @FocusState private var isAnswerFocused: Bool

    var body: some View {
        NavigationSplitView {
            SidebarView(viewModel: viewModel) {
                presentWordImporter()
            }
            .navigationSplitViewColumnWidth(min: 280, ideal: 320)
        } detail: {
            DictationWorkspace(viewModel: viewModel, isAnswerFocused: $isAnswerFocused)
        }
        .fileImporter(
            isPresented: $isImporterPresented,
            allowedContentTypes: [.plainText, .json],
            allowsMultipleSelection: false
        ) { result in
            switch result {
            case .success(let urls):
                guard let url = urls.first else { return }
                viewModel.importWords(from: url)
            case .failure(let error):
                viewModel.message = "导入失败：\(error.localizedDescription)"
            }
        }
        .alert("提示", isPresented: alertBinding) {
            Button("好", role: .cancel) {
                viewModel.message = nil
            }
        } message: {
            Text(viewModel.message ?? "")
        }
        .onChange(of: viewModel.status) { newValue in
            if newValue == .playing && viewModel.mode == .interactive {
                isAnswerFocused = true
            }
        }
    }

    private func presentWordImporter() {
        #if os(macOS)
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.plainText, .json]
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = false
        panel.canChooseFiles = true
        panel.title = "选择词单"
        panel.prompt = "导入"

        if panel.runModal() == .OK, let url = panel.url {
            viewModel.importWords(from: url)
        }
        #else
        isImporterPresented = true
        #endif
    }

    private var alertBinding: Binding<Bool> {
        Binding(
            get: { viewModel.message != nil },
            set: { isPresented in
                if !isPresented {
                    viewModel.message = nil
                }
            }
        )
    }
}

struct SidebarView: View {
    @ObservedObject var viewModel: DictationViewModel
    let importAction: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            VStack(alignment: .leading, spacing: 8) {
                Text("Word Dictation")
                    .font(.largeTitle.bold())
                Text("原生 SwiftUI 单词听写练习")
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 10) {
                StatCard(value: "\(viewModel.words.count)", label: "单词数")
                StatCard(value: "\(viewModel.mistakeCount)", label: "错误次数")
            }

            Button(action: importAction) {
                Label("导入词单", systemImage: "tray.and.arrow.down")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)

            Picker("练习模式", selection: $viewModel.mode) {
                ForEach(DictationMode.allCases) { mode in
                    Text(mode.title).tag(mode)
                }
            }
            .pickerStyle(.segmented)

            SettingsView(settings: $viewModel.settings)

            Divider()

            VStack(alignment: .leading, spacing: 10) {
                Text("词单预览")
                    .font(.headline)

                if viewModel.words.isEmpty {
                    EmptyStateView(
                        title: "暂无词单",
                        systemImage: "doc.text",
                        description: "支持 TXT 与 JSON 文件。"
                    )
                } else {
                    List(Array(viewModel.words.enumerated()), id: \.offset) { index, word in
                        HStack {
                            Text("\(index + 1).")
                                .foregroundStyle(.secondary)
                                .monospacedDigit()
                            Text(word)
                                .lineLimit(1)
                        }
                    }
                    .listStyle(.plain)
                }
            }
        }
        .padding(20)
    }
}

struct DictationWorkspace: View {
    @ObservedObject var viewModel: DictationViewModel
    var isAnswerFocused: FocusState<Bool>.Binding

    var body: some View {
        VStack(alignment: .leading, spacing: 22) {
            HeaderView(viewModel: viewModel)

            ControlStrip(viewModel: viewModel)

            ProgressView(value: viewModel.progressFraction)
                .progressViewStyle(.linear)

            Group {
                switch viewModel.mode {
                case .paper:
                    PaperModeView(viewModel: viewModel)
                case .interactive:
                    InteractiveModeView(viewModel: viewModel, isAnswerFocused: isAnswerFocused)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        }
        .padding(28)
        .background(AppColors.windowBackground)
    }
}

struct HeaderView: View {
    @ObservedObject var viewModel: DictationViewModel

    var body: some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: 6) {
                Text(viewModel.mode.title)
                    .font(.title.bold())
                Text(viewModel.progressText)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            Label(viewModel.status.title, systemImage: statusIcon)
                .font(.headline)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(.quaternary, in: RoundedRectangle(cornerRadius: 8))
        }
    }

    private var statusIcon: String {
        switch viewModel.status {
        case .stopped:
            "stop.circle"
        case .playing:
            "play.circle"
        case .paused:
            "pause.circle"
        }
    }
}

struct ControlStrip: View {
    @ObservedObject var viewModel: DictationViewModel

    var body: some View {
        HStack(spacing: 10) {
            Button {
                viewModel.start()
            } label: {
                Label("开始", systemImage: "play.fill")
            }
            .buttonStyle(.borderedProminent)
            .disabled(!viewModel.canStart)

            Button {
                viewModel.togglePause()
            } label: {
                Label(viewModel.status == .paused ? "继续" : "暂停", systemImage: viewModel.status == .paused ? "play" : "pause")
            }
            .disabled(!viewModel.canControlPlayback)

            Button(role: .destructive) {
                viewModel.stop()
            } label: {
                Label("结束", systemImage: "stop.fill")
            }
            .disabled(!viewModel.canControlPlayback)

            Button {
                viewModel.shuffleWords()
            } label: {
                Label("打乱", systemImage: "shuffle")
            }
            .disabled(viewModel.status != .stopped || viewModel.words.isEmpty)

            Spacer()
        }
        .buttonStyle(.bordered)
        .controlSize(.large)
    }
}

struct PaperModeView: View {
    @ObservedObject var viewModel: DictationViewModel

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(viewModel.feedback)
                .font(.headline)
                .foregroundStyle(.secondary)

            if viewModel.words.isEmpty {
                EmptyStateView(
                    title: "等待导入词单",
                    systemImage: "speaker.wave.2",
                    description: "导入后点击开始即可自动朗读。"
                )
            } else {
                List(Array(viewModel.words.enumerated()), id: \.offset) { index, word in
                    HStack(spacing: 12) {
                        Text("\(index + 1).")
                            .foregroundStyle(.secondary)
                            .frame(width: 42, alignment: .trailing)
                            .monospacedDigit()
                        Text(word)
                        Spacer()
                        if viewModel.status != .stopped && index == viewModel.currentIndex {
                            Image(systemName: "speaker.wave.2.fill")
                                .foregroundStyle(.blue)
                        }
                    }
                    .padding(.vertical, 4)
                }
                .listStyle(.inset)
            }
        }
    }
}

struct InteractiveModeView: View {
    @ObservedObject var viewModel: DictationViewModel
    var isAnswerFocused: FocusState<Bool>.Binding

    var body: some View {
        VStack(spacing: 18) {
            Spacer(minLength: 20)

            Text(viewModel.feedback)
                .font(.title3.weight(.semibold))
                .multilineTextAlignment(.center)
                .foregroundStyle(feedbackColor)
                .frame(maxWidth: 520)

            TextField("输入拼写后按回车提交", text: $viewModel.answer)
                .textFieldStyle(.roundedBorder)
                .font(.title2)
                .multilineTextAlignment(.center)
                .focused(isAnswerFocused)
                .disabled(viewModel.status != .playing)
                .onSubmit {
                    viewModel.submitAnswer()
                }
                .frame(maxWidth: 520)

            HStack(spacing: 10) {
                Button {
                    viewModel.submitAnswer()
                } label: {
                    Label("提交", systemImage: "checkmark")
                }
                .keyboardShortcut(.return, modifiers: [])
                .disabled(viewModel.status != .playing)

                Button {
                    viewModel.replayCurrentWord()
                } label: {
                    Label("重听", systemImage: "speaker.wave.2")
                }
                .disabled(viewModel.status == .stopped || viewModel.words.isEmpty)

                Button {
                    viewModel.skipWord()
                } label: {
                    Label("跳过", systemImage: "forward")
                }
                .disabled(viewModel.status != .playing)
            }
            .buttonStyle(.bordered)
            .controlSize(.large)

            Spacer()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var feedbackColor: Color {
        if viewModel.feedback.contains("正确") || viewModel.feedback.contains("完成") {
            return .green
        }
        if viewModel.feedback.contains("错误") || viewModel.feedback.contains("跳过") {
            return .red
        }
        return .secondary
    }
}

struct SettingsView: View {
    @Binding var settings: DictationSettings

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("播放设置")
                .font(.headline)

            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Label("语速", systemImage: "speedometer")
                    Spacer()
                    Text(String(format: "%.2f", settings.speechRate))
                        .foregroundStyle(.secondary)
                        .monospacedDigit()
                }
                Slider(value: $settings.speechRate, in: 0.25...0.65, step: 0.05)
            }

            Stepper(value: $settings.gapSeconds, in: 0.5...8, step: 0.5) {
                Label("间隔 \(String(format: "%.1f", settings.gapSeconds)) 秒", systemImage: "timer")
            }

            Stepper(value: $settings.repeatCount, in: 1...3) {
                Label("重复 \(settings.repeatCount) 次", systemImage: "repeat")
            }
        }
        .padding(14)
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 8))
    }
}

struct StatCard: View {
    let value: String
    let label: String

    var body: some View {
        VStack(spacing: 4) {
            Text(value)
                .font(.title2.bold())
                .monospacedDigit()
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 12)
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 8))
    }
}

struct EmptyStateView: View {
    let title: String
    let systemImage: String
    let description: String

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: systemImage)
                .font(.system(size: 36))
                .foregroundStyle(.secondary)
            Text(title)
                .font(.headline)
            Text(description)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity, minHeight: 180)
        .padding()
    }
}

enum AppColors {
    static var windowBackground: Color {
        #if os(macOS)
        Color(nsColor: .windowBackgroundColor)
        #else
        Color(uiColor: .systemBackground)
        #endif
    }
}
