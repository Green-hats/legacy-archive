# Word Dictation / 英语单词听写练习工具

一个用于英语单词听写和拼写练习的个人工具项目。当前以 SwiftUI 原生版本为主，网页版本保留在根目录 `index.html` 中。

## SwiftUI 版本

SwiftUI 应用位于 `SwiftUIApp/`，使用 Swift Package 管理，无需额外依赖。

### 运行方式

```bash
cd SwiftUIApp
swift run
```

也可以用 Xcode 直接打开 `SwiftUIApp/Package.swift`，选择 `WordDictationSwiftUI` 运行。

### 功能

- 支持纸质模式：自动朗读词单，结束后显示完整答案。
- 支持在线模式：听单词后输入拼写，自动判断正误。
- 支持导入 `.txt` 和 `.json` 词单。
- 支持语速、朗读间隔、重复次数设置。
- 支持重听、跳过、暂停、继续、结束和打乱顺序。
- 自动去重，避免重复单词干扰练习。
- 使用系统 `AVSpeechSynthesizer` 进行英文朗读。

## 词单格式

TXT 文件可使用换行、逗号或分号分隔：

```txt
apple
banana
computer, dictionary; education
```

JSON 文件支持字符串数组：

```json
["apple", "banana", "computer"]
```

也支持对象数组，对象中可使用 `word`、`en` 或 `text` 字段：

```json
[
  { "word": "apple" },
  { "en": "banana" },
  { "text": "computer" }
]
```

## 网页版本

根目录的 `index.html` 是保留的网页版本，可以直接用浏览器打开。网页版本使用浏览器内置 Web Speech API，建议使用最新版 Chrome、Edge 或 Safari。

## 项目结构

```text
Word-Dictation/
├── SwiftUIApp/   # SwiftUI 原生应用
├── index.html    # 网页版本
└── README.md
```
