// OmniLLM console. Everything the server returns is shown with textContent;
// nothing from the server or from a device is ever inserted as HTML.
"use strict";

// Words for people, the same in every OmniLLM screen: the hub is "your Mac"
// (here: this Mac; "this computer" when it is not a Mac, see HOST), the
// relay is "remote access", a provider is a "model source". Text in
// `backticks` is shown in code style (see rich()). A key with an "Other"
// twin (hubDown, hubDownOther) has words that hold only on a Mac (the menu
// bar, the keychain): the twin is used elsewhere.
const T = {
  zh: {
    loading: "正在载入……",
    loginTitle: "需要重新打开控制台",
    loginBody: "点屏幕顶部菜单栏里的 OmniLLM 图标，选“打开控制台”。", loginBodyOther: "在终端运行：",
    loginOr: "或者在终端运行：", loginOrOther: "",
    loginSafe: "为了安全，控制台只能从{here}上打开。",
    hubDown: "OmniLLM 没有运行（正在更新的话，过一会儿会自己回来）。点菜单栏里的 OmniLLM 图标，选“启动 OmniLLM”，或在终端运行 omnillm install。",
    hubDownOther: "OmniLLM 没有运行。在终端运行 omnillm install。",
    ariaLang: "切换语言", ariaTheme: "切换深色或浅色", ariaSteps: "步骤", back: "返回",
    step1: "接入模型", step2: "添加设备", step3: "试一下",
    w1Title: "先接入一个模型",
    w1Claude: "我有 Claude 订阅",
    w1Codex: "我有 ChatGPT 订阅",
    w1Agy: "我有 Google AI 订阅（Gemini）", subPlanAgy: "用你的 Google 账号登录；Google AI Pro、Ultra 订阅的额度更多。",
    subSignedOutAgy: "{1} 已经装好。还要给 OmniLLM 单独登录一次（你自己用的不受影响）。",
    subOn: "已接入", subOnBody: "可以用了。", subLoginLost: "{1} 的登录失效了，重新登录就能接着用。",
    subDesktop: "发现 {0}。同一个订阅也能给手机用：再装上官方命令行版 {1}（安装不收费，约 1 分钟），用同一个账号登录就好。{2}",
    subNone: "在{here}上装好官方的 {1}（安装不收费，约 1 分钟），用你的订阅账号登录。{2}",
    subPlanClaude: "需要 Claude Pro 或 Max 订阅。", subPlanCodex: "需要 ChatGPT Plus、Pro 等付费订阅。",
    subSignedOut: "{1} 已经装好，还没登录。",
    subPending: "{1} 已登录，正在接入……",
    subElsewhere: "Codex 已经登录，但这次的登录 OmniLLM 用不上。点“登录”，用同一个账号再登一次就好。",
    subOpenAgain: "重新打开", subCmdHere: "在{here}的终端里运行这一行：",
    subRemoved: "{1} 已登录，之前被你从这里移除了。",
    subInstall: "安装并登录", subSignIn: "登录", subReconnect: "重新接入",
    subHow: "会打开一个终端窗口，每一步都写在里面；登录在浏览器里完成。",
    subWaiting: "在打开的终端窗口和浏览器里完成登录；好了以后这里会自己接上。",
    subWaitingLong: "还没好？看一下终端窗口里写了什么。窗口没打开的话，在终端里运行下面这一行。",
    subConnected: "已接入 {0}。",
    w1Key: "我有 API Key",
    w1Free: "想免费：安装 Ollama，用{here}上的模型。",
    recheck: "重新检查", recheckNone: "还没发现已登录的订阅，也没发现{here}上的模型。", recheckNoMore: "没有发现别的已登录订阅或{here}上的模型。", connectedNames: "已接入 {0}。",
    srcClaude: "Claude 订阅", srcCodex: "ChatGPT 订阅（Codex）", srcAgy: "Gemini 订阅（Antigravity）", srcOllama: "Ollama（{here}）", srcLocal: "{here}上的模型（{0}）", srcFailed: "{0}：{1}",
    ollamaFound: "发现了{here}上的本地模型（Ollama）",
    ollamaUse: "使用{here}上的模型",
    source: "来源", otherCompatible: "其他 OpenAI 兼容服务",
    cname: "名字（英文，不要空格，如 groq）", curl: "接口地址（https://）",
    key: "API Key",
    keyHint: "会先用它做一次最小的请求来确认能用，成功才保存；只存在{here}的钥匙串里。",
    keyHintOther: "会先用它做一次最小的请求来确认能用，成功才保存；只存在{here}的密钥环里。",
    connect: "验证并接入",
    w2Title: "添加一台设备",
    skipPair: "先跳过，以后再加",
    w3Title: "试一下", pairedTry: "好了：设备已添加。在它上面发第一条消息试试。",
    model: "模型", message: "消息", send: "发送", finish: "完成",
    todaySpend: "今天的花费", estimate: "API Key 按官方价格估算，订阅和{here}上的模型不花钱", devices: "设备", models: "模型", apps: "应用", usage: "用量",
    addDevice: "添加设备", addProvider: "接入模型", addApp: "添加应用", add: "添加",
    coTitle: "让“{0}”只能手动对话？", coBody: "打开后，下面这些就用不了“{0}”了：",
    coApp: "应用", coDevice: "只给程序用的设备", coOnly: "它只有这一个能用的模型，会完全停下来",
    coYes: "仍然打开", coHow: "以后要让它们再用上，把这个开关关掉就行。",
    name: "名字", lastSeen: "最近在线", budget: "每天额度", todayBudget: "今天 / 每天额度", where: "对话发往", humanOnly: "只能手动对话",
    showOnDevices: "在设备上显示", inAuto: "Auto 可选", inAutoTip: "Auto 按问题自动选模型时，可以选它", subAlways: "订阅模型始终只能手动对话", editModel: "编辑模型 · {0}",
    enabled: "开启", pausedChip: "已暂停", pausedToast: "已暂停“{0}”", sourceOffToast: "已关闭“{0}”", sourceOnToast: "已打开“{0}”", resumedToast: "已开启“{0}”",
    online: "在线", todayAt: "今天 {0}", yesterdayAt: "昨天 {0}",
    hcAdded: "添加于", hcSeen: "最近在线", hcToday: "今天用量", hcModels: "能用的模型", hcOffline: "离线", hcKind: "类型",
    hcBackground: "只给程序用", hcApp: "这台电脑上的程序",
    hcAllTitle: "全部模型 · {0} 个", hcAllNote: "以后接入的新模型也自动可用。", hcSomeTitle: "能用 {0} 个，共 {1} 个",
    hcOff: "没开", hcNoneUsable: "现在没有它能用的模型。",
    editDevice: "编辑设备 · {0}", editApp: "编辑应用 · {0}", secStatus: "状态", secModels: "可用模型",
    infoAdded: "{0} 添加", infoSeen: "最近在线：{0}", infoSpent: "今天已用 {0}",
    pauseHint: "关掉即暂停：这台设备马上停用，配对和设置都保留。", pauseHintApp: "关掉即暂停：这个应用马上停用，令牌和设置都保留。",
    limitHint: "只受{whole}每天总额度（{0}）限制。", limitHintNone: "{whole}也没有总额度。", allModels: "全部模型（以后接入的新模型也自动可用）",
    grpSub: "订阅", grpKey: "API Key", grpLocal: "{Here}", pickOne: "至少选一个模型。", appNoHuman: "应用用不了只能手动对话的模型",
    hiddenTag: "不在设备上显示", sourceOffTag: "来源已关闭", appNoModels: "没有能用的模型", modelsCol: "模型", modelsAll: "全部 {0} 个", modelsSome: "部分 {0} / {1}", appNoModelsBody: "现在接入的模型应用都用不了。先接入 API Key 或{here}上的模型。",
    appNoModelsTip: "接入 API Key 或{here}上的模型后就能用", devNoModelsBody: "它能用的模型都已移除或不在设备上显示。打开“全部模型”，或在下面选几个。",
    devNoModelsBg: "这台设备只给程序和脚本用，可它能选的模型都设成了“只能手动对话”。在下面换一个模型，或在“模型”里关掉那个模型的“只能手动对话”。",
    removeDevice: "移除设备", removeApp: "移除应用",
    discardQ: "放弃没保存的修改？", discard: "放弃", keepEditing: "继续编辑",
    providers: "模型来源", keyLast4: "接入方式", subLogin: "订阅登录（只留在{here}上）",
    problemLogin: "登录已失效：在终端运行 omnillm login claude", problemLoginOther: "登录已失效：在终端里运行 omnillm login claude",
    problemLoginCodex: "登录已失效：在终端运行 omnillm login chatgpt", problemLoginAgy: "登录已失效：在终端运行 omnillm login gemini", problemLoginCodexOther: "登录已失效：在终端里运行 omnillm login chatgpt",
    problemLimit: "用量到上限，约 {0}后恢复", problemLimitNow: "用量到上限，稍后恢复", hours: "{0} 小时", hour: "1 小时", minutes: "{0} 分钟", minute: "1 分钟",
    noDevices: "还没有设备。点“添加设备”。", deviceCount: "{0} 台设备", deviceCountOne: "1 台设备", always: "始终",
    appsBody: "供脚本和程序后台调用，仅可使用 API Key 和{here}上的模型。",
    appsHelp: "接入方式", apAddr: "接口地址", apKey: "Key", apKeyV: "应用的令牌", apFmt: "格式", apFmtV: "兼容 OpenAI，也支持 Anthropic",
    apWho: "可连接", apWhoV: "仅{here}上的程序",
    appsOff: "给应用用的本机接口没能打开，应用暂时连不上。如果不是你特意关掉的，多半是别的程序占用了它的端口 8765：关掉那个程序后，在终端运行 omnillm install。",
    appTokenUse: "在程序里这样设置（大多数工具认这两个变量；也可以在设置里填同样的地址和 Key）：",
    appTokenTry: "试一下（应该列出它能用的模型）：",
    usageBody: "按{here}的时间计。花费按公布的价格估算，没有公布价格的按偏高估；订阅和{here}上的模型不花钱。",
    rangeToday: "今天", rangeYesterday: "昨天", rangeWeek: "最近 7 天",
    totalToday: "今天合计", totalYesterday: "昨天合计", totalWeek: "最近 7 天合计",
    emptyToday: "今天还没有用过。", emptyYesterday: "昨天没有用过。", emptyWeek: "最近 7 天没有用过。",
    byModel: "按模型", modelsGone: "已移除的模型", byModelEarlier: "没记下是哪个模型",
    who: "设备或应用", spend: "花费", calls: "次数", tokens: "Token（发出 / 收到）",
    usageHost: "{Here}", usageGone: "已移除的设备和应用", usageGoneShort: "已移除", goneCount: "（{0} 个）", usageGoneOne: "{0}（已移除）",
    capLine: "{whole}每天总额度：{0}", capLabel: "{whole}每天总额度", capNone: "{whole}每天总额度：不设上限", capEdit: "修改", noApps: "还没有应用。",
    modelsBody: "“只能手动对话”：仅限人工对话，脚本与应用无法调用。订阅模型强制开启，以防账号被判定为自动化使用。",
    codexName: "ChatGPT（Codex · 最新版）", codexVer: "ChatGPT（Codex · {0}）", latest: "（最新版）",
    remove: "移除", removeConfirm: "移除“{0}”？它会立刻用不了。", removeConfirmSource: "移除“{0}”？它的模型马上从所有设备和应用里消失；用 API Key 接入的，保存的 Key 也会删掉。要再用得重新接入。只想停用的话，关掉它的开关就行。",
    removeConfirmDevice: "移除“{0}”？它会马上断开，那台设备上的对话也会清除。要再用，得重新添加。",
    removed: "已移除", connected: "已接入，{0} 个模型可用", connectedOne: "已接入，1 个模型可用", saved: "已保存",
    local: "只在{here}上", remote: "云端", localChip: "{Here}", sub: "订阅",
    todayOf: "{0} / {1}", todayNoLimit: "{0} · 不设上限",
    pairTabPhone: "手机或平板", pairTabComputer: "电脑或服务器",
    pairP1: "用手机相机扫这个码", pairP1d: "iPhone 用相机，安卓用相机或浏览器扫码，不用装 App。",
    pairP2: "手机上点“添加”", pairP2d: "会打开 OmniLLM 网页，问你要不要添加到{here}。",
    pairL1: "在那台电脑的浏览器里打开链接", pairL1d: "复制链接，自己带到那台电脑上打开。链接相当于钥匙，别发给别人。",
    pairL2: "那台电脑上点“添加”", pairL2d: "会打开 OmniLLM 网页，问你要不要添加到{here}。",
    pairK1: "在那台电脑的终端里运行下面的命令", pairK1d: "已经装了 OmniLLM 也运行这条，不用 sudo。",
    pairK2: "它要配对文字时，粘贴下面这段", pairK2d: "配对文字相当于钥匙，别发给别人。",
    pairS3: "核对 6 位数", pairS3d: "两边会出现同样的 6 位数，一样就在这里点“一致，添加”。",
    pairWaitPhone: "等你用手机扫码……", pairWaitLink: "等那台电脑打开链接……", pairWaitCli: "等那台电脑连上……",
    pairConfirmQ: "“{0}”请求加入。它屏幕上的数字和这里的一样吗？", pairHere: "{Here}上的数字",
    pairExpiresIn: "{0} 后过期 · 只能用一次", copyLink: "复制链接", linkCopied: "已复制，去那台电脑打开", copyFailed: "复制不了，请用二维码或命令行",
    pairCliHero: "在那台电脑的终端里连接", pairCliToggle: "那台电脑没有浏览器？用命令行",
    pairCmd1: "在那台电脑的终端里运行", pairCmd1d: "已经装了也运行这条，不用 sudo",
    pairCmd2: "它要配对文字时，粘贴这段，回车", pairCmd2d: "粘贴后终端里看不到字，是正常的",
    pairYes: "一致，添加", pairNo: "不一致，不添加", pairLeft: "那台设备取消了，或者连接断了。",
    pairDone: "已添加“{0}”", pairFailed: "配对没有完成：{0}", pairRelayDrop: "远程连接断了，这次没有添加。网络恢复后再换一个。",
    pairDenied: "没有添加：你选了不一致。", pairExpired: "二维码过期了", pairExpiredLink: "链接过期了", pairExpiredText: "配对文字过期了", pairEnded: "这次没有添加",
    pairExpiredSay: "过期了，换一个就行。", pairNew: "换一个二维码", pairNewLink: "换一个链接", pairNewText: "换一段配对文字", qrAlt: "添加设备的二维码",
    pairStartOffline: "远程连接现在没有连上，生成不了二维码。", retry: "重试",
    close: "关闭", edit: "编辑", save: "保存", cancel: "取消", noLimit: "不设上限",
    budgetLabel: "每天额度（美元）",
    noRelay: "还没有设置远程连接，其他设备暂时连不上。可以先跳过；设置好远程连接后，在“设备”里添加。",
    relayWait: "正在打开远程连接……", relayDown: "远程连接现在连不上，请检查{here}的网络。连上之后这里会自动继续。",
    relayOff: "远程连接断开，设备暂时连不上", relayNone: "只在{here}上用（没有设置远程连接）",
    regionPaused: "{Here}所在的地区变了，已暂停使用模型。", webBad: "警告：手机用的网页和 GitHub 上签名发布的不一致，先别在手机上用。详情运行 omnillm verify web", regionConfirm: "确认继续",
    appTokenFor: "“{0}”的令牌", appTokenOnce: "只显示这一次，现在复制。",
    appName: "应用的名字",
    tryText: "你好",
    never: "还没用过", copy: "复制", copied: "已复制", undo: "撤销", oops: "出了点问题，请再试一次。",
  },
  en: {
    loading: "Loading…",
    loginTitle: "Open the console again",
    loginBody: "Click the OmniLLM icon in the menu bar and choose “Open the console”.", loginBodyOther: "Run in a terminal:",
    loginOr: "Or run in Terminal:", loginOrOther: "",
    loginSafe: "For safety, the console opens only from {here}.",
    hubDown: "OmniLLM isn’t running (if it’s updating, it comes back by itself shortly). Click the OmniLLM icon in the menu bar and choose “Start OmniLLM”, or run omnillm install in Terminal.",
    hubDownOther: "OmniLLM isn’t running. Run omnillm install in a terminal.",
    ariaLang: "Language", ariaTheme: "Light or dark", ariaSteps: "Steps", back: "Back",
    step1: "Connect a model", step2: "Add a device", step3: "Try it",
    w1Title: "First, connect a model",
    w1Claude: "I have a Claude subscription",
    w1Codex: "I have a ChatGPT subscription",
    w1Agy: "I have a Google AI subscription (Gemini)", subPlanAgy: "Sign in with your Google account; Google AI Pro and Ultra plans get more usage.",
    subSignedOutAgy: "{1} is installed. Sign in once more, for OmniLLM only (the one you use yourself isn’t affected).",
    subOn: "Connected", subOnBody: "Ready to use.", subLoginLost: "{1} is signed out; sign in again to keep using it.",
    subDesktop: "Found {0}. The same subscription can serve your phone too: add {1}, the official command-line program (no charge to install, about a minute), and sign in with the same account. {2}",
    subNone: "Install {1}, the official program, on {here} (no charge to install, about a minute) and sign in with your subscription account. {2}",
    subPlanClaude: "Needs a Claude Pro or Max plan.", subPlanCodex: "Needs a paid ChatGPT plan such as Plus or Pro.",
    subSignedOut: "{1} is installed but not signed in.",
    subPending: "{1} is signed in; connecting…",
    subElsewhere: "Codex is signed in, but OmniLLM can’t use that sign-in. Choose “Sign in” and sign in once more with the same account.",
    subOpenAgain: "Open again", subCmdHere: "In a terminal on {here}, run:",
    subRemoved: "{1} is signed in; you removed it from here earlier.",
    subInstall: "Install and sign in", subSignIn: "Sign in", subReconnect: "Connect again",
    subHow: "A Terminal window opens with every step written out; you sign in in your browser.",
    subWaiting: "Finish signing in in the Terminal window and your browser; this connects by itself when you’re done.",
    subWaitingLong: "Not yet? Check what the Terminal window says. If no window opened, run the line below in a terminal.",
    subConnected: "Connected your {0}.",
    w1Key: "I have an API key",
    w1Free: "Want it free? Install Ollama to use models on {here}.",
    recheck: "Check again", recheckNone: "No signed-in subscription or model on {here} found yet.", recheckNoMore: "No other signed-in subscription or model on {here} found.", connectedNames: "Connected {0}.",
    srcClaude: "Claude subscription", srcCodex: "ChatGPT subscription (Codex)", srcAgy: "Google AI subscription (Antigravity)", srcOllama: "Ollama ({here})", srcLocal: "Model on {here} ({0})", srcFailed: "{0}: {1}",
    ollamaFound: "A local model (Ollama) is running on {here}",
    ollamaUse: "Use models on {here}",
    source: "Source", otherCompatible: "Other OpenAI-compatible service",
    cname: "Name (no spaces, e.g. groq)", curl: "API address (https://)",
    key: "API key",
    keyHint: "We check it with one tiny request and save it only if it works. It’s kept only in {here}’s keychain.",
    keyHintOther: "We check it with one tiny request and save it only if it works. It’s kept only in {here}’s keyring.",
    connect: "Check and connect",
    w2Title: "Add a device",
    skipPair: "Skip for now",
    w3Title: "Try it", pairedTry: "All set: the device is added. Send your first message from it.",
    model: "Model", message: "Message", send: "Send", finish: "Done",
    todaySpend: "Spent today", estimate: "API keys at their official prices; subscriptions and models on {here} cost nothing", devices: "Devices", models: "Models", apps: "Apps", usage: "Usage",
    addDevice: "Add device", addProvider: "Connect a model", addApp: "Add app", add: "Add",
    coTitle: "Make “{0}” chat only?", coBody: "Once it’s on, these can no longer use “{0}”:",
    coApp: "App", coDevice: "Programs-only device", coOnly: "its only usable model: it stops working entirely",
    coYes: "Turn on anyway", coHow: "To let them use it again later, turn this switch off.",
    name: "Name", lastSeen: "Last seen", budget: "Daily limit", todayBudget: "Today / daily limit", where: "Messages go to", humanOnly: "Chat only",
    showOnDevices: "Show on devices", inAuto: "In Auto", inAutoTip: "Auto may choose it when it picks a model for each message", subAlways: "Subscription models are always chat-only", editModel: "Edit model · {0}",
    enabled: "On", pausedChip: "Paused", pausedToast: "Paused “{0}”", sourceOffToast: "Turned off “{0}”", sourceOnToast: "Turned on “{0}”", resumedToast: "Turned on “{0}”",
    online: "Online", todayAt: "Today {0}", yesterdayAt: "Yesterday {0}",
    hcAdded: "Added", hcSeen: "Last seen", hcToday: "Today", hcModels: "Models", hcOffline: "Offline", hcKind: "Kind",
    hcBackground: "Programs only", hcApp: "A program on this computer",
    hcAllTitle: "All models · {0}", hcAllNote: "Models connected later are included too.", hcSomeTitle: "{0} of {1} models",
    hcOff: "Off", hcNoneUsable: "No model it can use right now.",
    editDevice: "Edit device · {0}", editApp: "Edit app · {0}", secStatus: "Status", secModels: "Models it can use",
    infoAdded: "Added {0}", infoSeen: "Last seen: {0}", infoSpent: "Spent today {0}",
    pauseHint: "Turn off to pause: the device stops at once; its pairing and settings are kept.", pauseHintApp: "Turn off to pause: the app stops at once; its token and settings are kept.",
    limitHint: "Only the daily total for {here} ({0}) applies.", limitHintNone: "{Here} has no daily total either.", allModels: "All models (including ones added later)",
    grpSub: "Subscriptions", grpKey: "API key", grpLocal: "{Here}", pickOne: "Choose at least one model.", appNoHuman: "Apps can’t use chat-only models",
    hiddenTag: "Hidden on devices", sourceOffTag: "Source off", appNoModels: "No usable models", modelsCol: "Models", modelsAll: "All {0}", modelsSome: "{0} of {1}", appNoModelsBody: "None of the connected models work for apps. Connect an API key or a model on {here} first.",
    appNoModelsTip: "Works once you connect an API key or a model on {here}", devNoModelsBody: "All the models it could use were removed or hidden on devices. Turn on “All models” or choose some below.",
    devNoModelsBg: "This device is for programs and scripts only, but every model it may use is set to “Chat only”. Choose another model below, or turn off “Chat only” for that model under “Models”.",
    removeDevice: "Remove device", removeApp: "Remove app",
    discardQ: "Discard unsaved changes?", discard: "Discard", keepEditing: "Keep editing",
    providers: "Model sources", keyLast4: "Connected with", subLogin: "Subscription login (stays on {here})",
    problemLogin: "Signed out: in Terminal, run omnillm login claude", problemLoginOther: "Signed out: in a terminal, run omnillm login claude",
    problemLoginCodex: "Signed out: in Terminal, run omnillm login chatgpt", problemLoginAgy: "Signed out: in Terminal, run omnillm login gemini", problemLoginCodexOther: "Signed out: in a terminal, run omnillm login chatgpt",
    problemLimit: "Usage limit reached; back in about {0}", problemLimitNow: "Usage limit reached; back soon", hours: "{0} hours", hour: "1 hour", minutes: "{0} minutes", minute: "1 minute",
    noDevices: "No devices yet. Choose “Add device”.", deviceCount: "{0} devices", deviceCountOne: "1 device", always: "Always",
    appsBody: "For scripts and programs: background calls only, with API keys and models on {here}.",
    appsHelp: "How to connect", apAddr: "Address", apKey: "Key", apKeyV: "The app’s token", apFmt: "Format", apFmtV: "OpenAI-compatible; Anthropic works too",
    apWho: "Reachable from", apWhoV: "Programs on {here} only",
    appsOff: "The local API for apps isn’t on, so apps can’t connect. Unless you turned it off on purpose, another program is probably using its port (8765): close it, then run omnillm install in Terminal.",
    appTokenUse: "Set it up in the program (most tools read these two variables; or enter the same address and key in its settings):",
    appTokenTry: "Try it (it should list the models it can use):",
    usageBody: "Days follow {here}’s clock. Spend is estimated from published prices (on the high side where none is published); subscriptions and models on {here} cost nothing.",
    rangeToday: "Today", rangeYesterday: "Yesterday", rangeWeek: "Last 7 days",
    totalToday: "Total today", totalYesterday: "Total yesterday", totalWeek: "Total, last 7 days",
    emptyToday: "Nothing used yet today.", emptyYesterday: "Nothing was used yesterday.", emptyWeek: "Nothing used in the last 7 days.",
    byModel: "By model", modelsGone: "Removed models", byModelEarlier: "Model not recorded",
    who: "Device or app", spend: "Spend", calls: "Calls", tokens: "Tokens (in / out)",
    usageHost: "{Here}", usageGone: "Removed devices and apps", usageGoneShort: "Removed", goneCount: "({0})", usageGoneOne: "{0} (removed)",
    capLine: "Daily total for {here}: {0}", capLabel: "Daily total for {here}", capNone: "Daily total for {here}: no limit", capEdit: "Change", noApps: "No apps yet.",
    modelsBody: "Chat-only models serve people only; scripts and apps can’t call them. Always on for subscriptions, so the account isn’t flagged as automated.",
    codexName: "ChatGPT (Codex · latest)", codexVer: "ChatGPT (Codex · {0})", latest: " (latest)",
    remove: "Remove", removeConfirm: "Remove “{0}”? It stops working at once.", removeConfirmSource: "Remove “{0}”? Its models disappear from every device and app at once, and a saved API key is deleted; to use it again you connect it again. To stop it for now, turn its switch off instead.",
    removeConfirmDevice: "Remove “{0}”? It disconnects at once and its chats on that device are cleared; to use it again, add it again.",
    removed: "Removed", connected: "Connected: {0} models available", connectedOne: "Connected: 1 model available", saved: "Saved",
    local: "{Here} only", remote: "Cloud", localChip: "{Here}", sub: "subscription",
    todayOf: "{0} / {1}", todayNoLimit: "{0} · no limit",
    pairTabPhone: "Phone or tablet", pairTabComputer: "Computer or server",
    pairP1: "Scan this code with your phone’s camera", pairP1d: "On iPhone use the Camera; on Android the camera or a browser. No app to install.",
    pairP2: "Tap “Add” on the phone", pairP2d: "The OmniLLM web page opens and asks to add the phone to {here}.",
    pairL1: "Open the link in that computer’s browser", pairL1d: "Copy the link and take it to that computer yourself. It works like a key: don’t send it to anyone.",
    pairL2: "Choose “Add” on that computer", pairL2d: "The OmniLLM web page opens and asks to add it to {here}.",
    pairK1: "Run the command below in that computer’s terminal", pairK1d: "Run it even if OmniLLM is already installed. No sudo.",
    pairK2: "When it asks for the pairing text, paste the text below", pairK2d: "The pairing text works like a key: don’t send it to anyone.",
    pairS3: "Check the 6 digits", pairS3d: "Both sides show the same 6 digits. If they match, choose “Matches, add” here.",
    pairWaitPhone: "Waiting for your phone to scan…", pairWaitLink: "Waiting for that computer to open the link…", pairWaitCli: "Waiting for that computer to connect…",
    pairConfirmQ: "“{0}” wants to join. Does its screen show the same number as here?", pairHere: "The number on {here}",
    pairExpiresIn: "Expires in {0} · single use", copyLink: "Copy link", linkCopied: "Copied: open it on that computer", copyFailed: "Can’t copy here; use the QR code or the command line",
    pairCliHero: "Connect from that computer’s terminal", pairCliToggle: "No browser on that computer? Use the command line",
    pairCmd1: "Run this in that computer’s terminal", pairCmd1d: "Run it even if OmniLLM is already installed. No sudo.",
    pairCmd2: "When it asks for the pairing text, paste this and press Return", pairCmd2d: "Nothing shows while you paste; that’s normal.",
    pairYes: "Matches, add", pairNo: "Doesn’t match, don’t add", pairLeft: "The device cancelled or lost its connection.",
    pairDone: "Added “{0}”", pairFailed: "Pairing didn’t finish: {0}", pairRelayDrop: "Remote access dropped, so this device wasn’t added. Once the network is back, make a new one.",
    pairDenied: "Not added: you said it doesn’t match.", pairExpired: "The QR code expired", pairExpiredLink: "The link expired", pairExpiredText: "The pairing text expired", pairEnded: "Not added this time",
    pairExpiredSay: "It expired; just make a new one.", pairNew: "New QR code", pairNewLink: "New link", pairNewText: "New pairing text", qrAlt: "QR code for adding a device",
    pairStartOffline: "Remote access isn’t connected right now, so no QR code can be made.", retry: "Retry",
    close: "Close", edit: "Edit", save: "Save", cancel: "Cancel", noLimit: "No limit",
    budgetLabel: "Daily limit (USD)",
    noRelay: "Remote access isn’t set up yet, so other devices can’t connect. Skip for now; add devices under Devices once it’s set up.",
    relayWait: "Turning on remote access…", relayDown: "Remote access can’t connect right now; check {here}’s network. It picks up again once connected.",
    relayOff: "Remote access is down; devices can’t connect now", relayNone: "{Here} only (remote access not set up)",
    regionPaused: "{Here}’s region changed; models are paused.", webBad: "Warning: the web page phones use differs from the signed GitHub release; don’t use it on phones for now. Details: omnillm verify web", regionConfirm: "Confirm and resume",
    appTokenFor: "Token for “{0}”", appTokenOnce: "Shown once — copy it now.",
    appName: "App name",
    tryText: "Hello",
    never: "Not yet", copy: "Copy", copied: "Copied", undo: "Undo", oops: "Something went wrong. Try again.",
  },
};

function store(k, v) { try { if (v === undefined) return localStorage.getItem(k); localStorage.setItem(k, v); } catch (_) { return null; } }

let lang = store("omnillm.lang") || ((navigator.language || "en").toLowerCase().startsWith("zh") ? "zh" : "en");
// The computer OmniLLM runs on, in the strings' {here} ("this Mac"), {Here}
// (at the start of a sentence) and {whole} ("整台 Mac"). The status says
// which system it is; before it arrives the browser's own system is the
// same one (the console opens only on that computer).
const HOST = {
  zh: { mac: { here: "这台 Mac", Here: "这台 Mac", whole: "整台 Mac" }, other: { here: "这台电脑", Here: "这台电脑", whole: "整台电脑" } },
  en: { mac: { here: "this Mac", Here: "This Mac", whole: "this Mac" }, other: { here: "this computer", Here: "This computer", whole: "this computer" } },
};
const onMac = () => (S && S.os ? S.os === "darwin" : /Mac/i.test(navigator.platform || navigator.userAgent || ""));
// hostWords puts the computer's words in; in Chinese a space separates
// "Mac" from the character after it ("这台 Mac 上"), none after "电脑".
function hostWords(s) {
  const mac = onMac(), w = HOST[lang][mac ? "mac" : "other"];
  return s.replace(/\{(here|Here|whole)\}([\u4e00-\u9fff]?)/g, (_, k, c) => w[k] + (c && mac && lang === "zh" ? " " : "") + c);
}
const t = (k, ...a) => {
  const d = T[lang], s = !onMac() && d[k + "Other"] !== undefined ? d[k + "Other"] : d[k];
  return hostWords(s === undefined ? k : s).replace(/\{(\d)\}/g, (_, i) => String(a[+i] ?? ""));
};
const $ = (id) => document.getElementById(id);
const el = (tag, text, cls) => { const e = document.createElement(tag); if (text !== undefined) e.textContent = text; if (cls) e.className = cls; return e; };
// composing: the key belongs to an input method (Chinese, Japanese…):
// Enter there picks the words and Escape drops them; neither saves,
// submits or closes anything. Safari sends the Enter that ends the
// composition as keyCode 229 with isComposing already false.
const composing = (e) => e.isComposing || e.keyCode === 229;
// Enter in a form's field submits it, also while an input method is
// choosing words in some browsers: those Enters don't.
function noImeSubmit(form) { form.addEventListener("keydown", (e) => { if (e.key === "Enter" && composing(e)) e.preventDefault(); }); }
// sourceName: a model source as people call it ("Claude subscription"),
// from its kind; null when it has no fixed name.
// Only the Ollama app itself is "Ollama"; another server on this Mac
// (llama.cpp, LM Studio) added under its own name goes by that name.
const sourceName = (kind, name) => kind === "claude-code" ? t("srcClaude") : kind === "codex" ? t("srcCodex") : kind === "antigravity" ? t("srcAgy")
  : kind === "ollama" ? (name && name !== "ollama" ? t("srcLocal", name) : t("srcOllama")) : null;

// rich sets text where `backticked` parts are commands, shown in code style.
// Still text only: the parts become text nodes and <code> elements.
function rich(node, text) {
  node.textContent = "";
  text.split("`").forEach((part, i) => { if (part) node.append(i % 2 ? el("code", part) : document.createTextNode(part)); });
  return node;
}

// copyBtn copies text and says so on the button.
function copyBtn(text, b) {
  b = b || el("button", t("copy"), "secondary small-btn copy");
  b.type = "button";
  b.addEventListener("click", async () => { try { await navigator.clipboard.writeText(text); b.textContent = t("copied"); } catch (_) { /* the block is user-select: all */ } });
  return b;
}

// codeBox: a command or pairing text with its Copy button in the top-right
// corner, so the button never sits between two blocks. A command (cmd)
// stays one line that scrolls sideways, never broken inside "--device";
// a token may break anywhere.
function codeBox(text, cmd) {
  const box = el("div", undefined, "codebox");
  box.append(el("div", text, cmd ? "code one-line" : "code"), copyBtn(text));
  return box;
}

// appliedMac: which system the page's fixed words were last put in for.
let appliedMac = null;
function applyLang() {
  appliedMac = onMac();
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  document.querySelectorAll("[data-t]").forEach((n) => { if (n.hasAttribute("data-rich")) rich(n, t(n.dataset.t)); else n.textContent = t(n.dataset.t); });
  document.querySelectorAll("[data-tip]").forEach((n) => { n.title = t(n.dataset.tip); });
  document.querySelectorAll("[data-aria]").forEach((n) => { n.setAttribute("aria-label", t(n.dataset.aria)); });
  $("lang").textContent = lang === "zh" ? "English" : "中文";
  const tt = $("try-text"); // only while it is still the default
  if (!tt.value || tt.value === T.zh.tryText || tt.value === T.en.tryText) tt.value = t("tryText");
}

function applyTheme(v) {
  if (v) document.documentElement.dataset.theme = v; else delete document.documentElement.dataset.theme;
}

// The session secret lives in this origin's sessionStorage (not shared
// with other localhost ports, unlike cookies) and goes with every call.
function sessionSecret(v) { try { if (v === undefined) return sessionStorage.getItem("omnillm.session") || ""; sessionStorage.setItem("omnillm.session", v); } catch (_) { return ""; } }

// Changes the owner makes (every POST): how many were sent, and how many
// are still on their way. A status poll that overlaps one is dropped, so
// it can't put back what was just changed (a pause switch flipping back).
let changes = 0, changing = 0;

async function api(path, body) {
  const opts = { method: body === undefined ? "GET" : "POST", credentials: "same-origin",
    headers: { "X-OmniLLM-Console": "1", "X-OmniLLM-Lang": lang, "X-OmniLLM-Session": sessionSecret() } };
  if (body !== undefined) { opts.headers["Content-Type"] = "application/json"; opts.body = JSON.stringify(body); changes++; changing++; }
  let r;
  // No answer at all (the fetch itself failed): OmniLLM isn't running. The
  // error has no status, which is how callers tell this case apart.
  try { r = await fetch("/console/api/" + path, opts); } catch (_) { throw new Error(t("hubDown")); } finally { if (body !== undefined) changing--; }
  let data = null;
  try { data = await r.json(); } catch (_) { data = null; }
  if (!r.ok) {
    // The hub's own sentence when it sent one; never "HTTP 502".
    const e = new Error((data && data.error && data.error.message) || t("oops"));
    e.status = r.status;
    e.code = data && data.error && data.error.code;
    // Any call that finds the session gone (the hub restarted, or the
    // session timed out) leads to the way back in, not to an error toast.
    if (r.status === 401 && path !== "login") loggedOut();
    throw e;
  }
  return data;
}

// loggedOut: the session is gone; stop everything that polls and show how
// to open the console again.
let signedOut = false;
function loggedOut() {
  signedOut = true;
  stopPair();
  $("hub-down").hidden = true;
  show("login-needed");
}

let toastTimer;
// toast says something for a moment (text, or the nodes of named());
// act ({ label, run }) adds a button (Undo) and keeps it up for 5 s.
function toast(msg, err, act, long) {
  if (signedOut) return; // the login panel already says what to do
  const n = $("toast");
  n.textContent = "";
  n.append(el("span", undefined, "toast-msg" + (Array.isArray(msg) ? " has-name" : "")));
  n.firstChild.append(...(Array.isArray(msg) ? msg : [msg]));
  n.className = "toast" + (err ? " err" : "");
  if (act) {
    const b = el("button", act.label, "toast-act"); b.type = "button";
    b.addEventListener("click", () => { clearTimeout(toastTimer); n.hidden = true; act.run(); });
    n.append(b);
  }
  n.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { n.hidden = true; }, err || long ? 6000 : act ? 5000 : 2500);
}

const money = (micro) => "$" + (micro / 1e6).toFixed(2);
// named: a sentence with a device's name in it ("Paused “…”"), the name a
// node of its own so a long one is cut short ("…") instead of the sentence.
function named(key, name) {
  const [a, b] = t(key, "\u0000").split("\u0000");
  return [a, el("span", name, "toast-name"), b];
}

let S = null;

// load asks for everything on the dashboard, with the Usage tab's range.
// The answer is tagged with that range (S.range) and dropped (null) when
// another range was picked meanwhile, or, for a poll, when the owner
// changed something while it was on its way.
async function load(poll) {
  const range = usageRange, before = changes, busyBefore = changing;
  const next = await api("status?range=" + range);
  if (range !== usageRange || (poll && (busyBefore || changing || before !== changes))) return null;
  for (const k of ["providers", "models", "devices", "apps", "usage", "range_usage", "model_usage"]) next[k] = next[k] || [];
  next.range = range;
  sortModels(next.models);
  return next;
}

async function refresh() {
  const next = await load(false);
  if (!next) return; // a newer range was asked for; its answer draws
  S = next;
  // The status says which system this is: the fixed words follow it.
  if (onMac() !== appliedMac) applyLang();
  // The owner's language, if chosen (here or in the menu bar).
  if ((S.lang === "zh" || S.lang === "en") && S.lang !== lang) { lang = S.lang; store("omnillm.lang", lang); applyLang(); }
  renderStatus();
  if (!wizardDone() && (S.providers.length === 0 || (S.devices.length === 0 && !store("omnillm.skipPair")))) {
    showWizard();
  } else {
    showDash();
  }
}

// The hub reaches the relay a moment after it starts, and reconnects by
// itself after a drop: keep the status line current instead of showing
// what was true when the page opened. A poll that gets no answer at all
// means the hub itself stopped: say so, and how to start it again.
setInterval(async () => {
  if (!S || signedOut || document.hidden) return;
  let r;
  try { r = await api("relay"); } catch (e) {
    if (e.status === undefined && $("hub-down").hidden) { $("hub-down").hidden = false; S.relay_up = false; renderStatus(); }
    return;
  }
  $("hub-down").hidden = true;
  followLang(r.lang);
  if (r.relay_up !== S.relay_up || r.relay_configured !== S.relay_configured) {
    S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
    renderStatus();
  }
}, 3000);

// Devices, usage and models change while the page is open (a phone pairs,
// someone chats): redraw the dashboard every 10 s, but never while
// something is being edited or a device is being added, so nothing typed
// or on screen is thrown away.
function busy() {
  // A hover card being read counts too: the redraw would take it away.
  return !!(curPair || pairTimer || hcOpen || document.querySelector("tr.editing, tr.asking, .cap-edit, .pair-panel, .dialog-scrim"));
}
setInterval(async () => {
  if (!S || signedOut || document.hidden || $("dash").hidden || busy()) return;
  let next;
  try { next = await load(true); } catch (_) { return; }
  if (!next || $("dash").hidden || busy()) return;
  S = next;
  if (followLang(S.lang)) return;
  renderStatus();
  showDash();
}, 10000);

function wizardDone() { return store("omnillm.wizard") === "done"; }

// renderStatus: the header line speaks only when something needs the
// owner: remote access down or not set up, or the region changed. While
// phones can connect it says nothing (that is the normal state).
function renderStatus() {
  const s = $("status");
  s.textContent = "";
  s.removeAttribute("title");
  // Left empty, not hidden: it is the spacer that keeps the tools right.
  // The phones' web page differs from the signed release: said first.
  const wc = S.web_check, webBad = wc && !wc.ok && !wc.error;
  if (S.relay_configured && S.relay_up && !(S.region && S.region.paused) && !webBad) return;
  const paused = S.region && S.region.paused;
  const text = [webBad ? t("webBad") : !S.relay_configured ? t("relayNone") : S.relay_up ? "" : t("relayOff"), paused ? t("regionPaused") : ""].filter(Boolean).join(" · ");
  // A warning (paused, the phones' page differs) is spelled out on a
  // phone-width window too, not left to a tooltip a touch screen can't show.
  s.classList.toggle("alert", !!(paused || webBad));
  // On a phone-width window only the dot shows; it carries the sentence
  // for screen readers and as a tooltip.
  const dot = el("span", "", "dot" + (S.relay_up && !webBad && !paused ? " on" : ""));
  dot.setAttribute("role", "img");
  dot.setAttribute("aria-label", text);
  s.title = text;
  const words = el("span", text, "status-text");
  words.setAttribute("aria-hidden", "true");
  s.append(dot, words);
  if (paused) {
    const b = el("button", t("regionConfirm"), "link");
    b.type = "button";
    b.addEventListener("click", async () => { try { await api("region/confirm", {}); await refresh(); } catch (e) { toast(e.message, true); } });
    s.append(document.createTextNode(" "), b);
  }
}

function show(id) { ["loading", "login-needed", "wizard", "dash"].forEach((x) => { $(x).hidden = x !== id; }); }

// ---------- wizard ----------

function stepTo(n) {
  [1, 2, 3].forEach((i) => {
    $("w" + i).hidden = i !== n;
    $("st" + i).className = i < n ? "done" : i === n ? "now" : "";
  });
}

// fromDash: "Connect a model" was opened from the dashboard, so the
// wizard offers a way back to it.
let fromDash = false;

// ---- Claude and ChatGPT on the first step ----
// One card each: what is done, and the one step left with the button that
// does it. The program's own installer and login run in a Terminal window
// the hub opens (the owner sees every command and signs in in the
// browser); while that runs the card asks the hub every 2 s and connects
// the subscription as soon as it is signed in.
const SUB = {
  "claude-code": { title: "w1Claude", app: "Claude", plan: "subPlanClaude", word: "claude", program: "Claude Code" },
  "codex": { title: "w1Codex", app: "ChatGPT", plan: "subPlanCodex", word: "chatgpt", program: "Codex" },
  "antigravity": { title: "w1Agy", app: "Antigravity", plan: "subPlanAgy", word: "gemini", program: "Antigravity CLI", signedOut: "subSignedOutAgy" },
};
// The desktop app by its name ("Claude", "ChatGPT", "Codex").
const desktopName = (n) => lang === "zh" ? n + " 桌面版" : "the " + n + " desktop app";
let subsTimer = 0, subsWaiting = {}, subsAsked = {}, subsCanOpen = true, subsSeq = 0;
const WAIT_MAX = 10 * 60e3;
async function renderSubs() {
  // Answers may come back out of order (a click, the timer, coming back
  // into view): only the newest one draws.
  const seq = ++subsSeq;
  let r;
  try { r = await api("subs"); } catch (_) { return; }
  if (seq !== subsSeq) return;
  for (const [k, since] of Object.entries(subsWaiting)) if (Date.now() - since > WAIT_MAX) delete subsWaiting[k];
  subsCanOpen = r.can_open;
  const box = $("subs");
  box.textContent = "";
  // The one nearest to usable first: connected, then signed in, installed,
  // a desktop app found, nothing.
  const rank = (x) => x.connected ? 0 : x.logged_in ? 1 : x.installed ? 2 : x.desktop ? 3 : 4;
  const subs = (r.subs || []).slice().sort((a, b) => rank(a) - rank(b));
  // Signed in from here after the owner had removed it: signing in was
  // asking for it again, so connect it without a second click.
  const again = subs.filter((x) => subsWaiting[x.kind] && x.logged_in && !x.connected && x.removed && !subsAsked[x.kind]);
  if (again.length) {
    for (const x of again) { subsAsked[x.kind] = true; try { await api("subs/connect", { kind: x.kind }); } catch (_) {} }
    return renderSubs();
  }
  let newly = [];
  for (const x of subs) {
    const k = SUB[x.kind];
    if (!k) continue;
    if (x.connected && !x.signed_out && subsWaiting[x.kind]) { newly.push(sourceName(x.kind) || k.app); delete subsWaiting[x.kind]; }
    box.append(subCard(x, k));
  }
  if (newly.length) toast(t("subConnected", newly.join(lang === "zh" ? "、" : ", ")));
  // Connected while the first step shows (a click, or one found signed
  // in): the wizard goes on to adding a device.
  if (newly.length || (subs.some((x) => x.connected) && S && S.providers.length === 0 && !fromDash)) refresh();
  // Keep asking while someone is signing in (at most 10 minutes).
  clearTimeout(subsTimer);
  // A signed-in one is being connected right now: look again shortly too.
  const pending = subs.some((x) => x.logged_in && !x.connected && !x.removed && !x.problem);
  if ((Object.keys(subsWaiting).length || pending) && !$("w1").hidden && !document.hidden) subsTimer = setTimeout(renderSubs, 3000);
}
function subCard(x, k) {
  const card = el("div", undefined, "choice sub-card");
  const head = el("div", undefined, "sub-head");
  head.append(el("h2", t(k.title)));
  // Connected but signed out (the Models tab says so): the card offers the sign-in.
  const signedOut = !!x.signed_out;
  // Switched off by the owner: said, not "Connected".
  const off = S && (S.providers || []).some((p) => p.kind === x.kind && p.off);
  if (x.connected && !signedOut) head.append(off ? el("span", t("sourceOffTag"), "tag warn-tag") : el("span", t("subOn"), "tag ok-tag"));
  card.append(head);
  const fill = (key) => t(key, desktopName(x.desktop || k.app), k.program, t(k.plan));
  if (x.connected && !signedOut) { delete subsWaiting[x.kind]; card.append(el("p", t("subOnBody"), "muted")); return card; }
  let text, btn = null, act = null;
  if (signedOut) { text = fill("subLoginLost"); btn = t("subSignIn"); act = "login"; }
  else if (x.logged_in && x.removed) { text = fill("subRemoved"); btn = t("subReconnect"); act = "connect"; }
  else if (x.logged_in && x.problem) { text = x.problem; btn = t("recheck"); act = "connect"; }
  else if (x.logged_in) { card.append(el("p", fill("subPending"), "muted")); return card; }
  else if (x.elsewhere) { text = t("subElsewhere"); btn = t("subSignIn"); act = "login"; }
  else if (x.installed) { text = fill(k.signedOut || "subSignedOut"); btn = t("subSignIn"); act = "login"; }
  else { text = fill(x.desktop ? "subDesktop" : "subNone"); btn = t("subInstall"); act = "login"; }
  card.append(el("p", text, "muted"));
  const cmd = "omnillm login " + k.word;
  if (act !== "login") delete subsWaiting[x.kind];
  const since = subsWaiting[x.kind];
  if (act === "login" && !subsCanOpen) {
    // No window to open here (not a Mac): the one line to run, and the
    // card keeps looking while the owner runs it.
    card.append(el("p", t("subCmdHere"), "small sub-cmd-label"), codeBox(cmd, true));
    if (!since) subsWaiting[x.kind] = Date.now();
    return card;
  }
  if (since && act === "login") {
    // Signing in now: say where it happens; after a while, the command
    // too. The button stays, to open the window again.
    const w = el("div", undefined, "sub-wait");
    w.append(el("span", "", "spinner sm"), el("span", t(Date.now() - since > 45e3 ? "subWaitingLong" : "subWaiting")));
    card.append(w);
    if (Date.now() - since > 45e3) card.append(codeBox(cmd, true));
    btn = t("subOpenAgain");
  }
  const b = el("button", btn, since && act === "login" ? "secondary" : "primary"); b.type = "button";
  b.addEventListener("click", async () => {
    b.disabled = true;
    try {
      if (act === "connect") {
        await api("subs/connect", { kind: x.kind });
        toast(t("subConnected", sourceName(x.kind) || k.app));
        await refresh();
      } else {
        await api("subs/login", { kind: x.kind });
        subsWaiting[x.kind] = Date.now();
        delete subsAsked[x.kind];
      }
    } catch (e) { toast(e.message, true); }
    renderSubs();
  });
  card.append(b);
  if (act === "login" && !since) card.append(el("p", t("subHow"), "muted small sub-how"));
  return card;
}
// Back in view (the owner signed in elsewhere meanwhile): look again now.
document.addEventListener("visibilitychange", () => { if (!document.hidden && !$("w1").hidden) renderSubs(); });

async function showWizard() {
  show("wizard");
  $("wiz-back-line").hidden = !fromDash;
  // From the dashboard: adding one more source, not the first-run steps.
  document.querySelector("#wizard .steps").hidden = fromDash;
  const h = $("w1").querySelector("h1");
  h.dataset.t = fromDash ? "addProvider" : "w1Title"; // kept through a language switch
  h.textContent = t(h.dataset.t);
  if (S.providers.length === 0 || fromDash) {
    renderSubs();
    stepTo(1);
    try {
      const o = await api("ollama");
      const chat = (o.models || []).filter((m) => !/embed|bge|nomic|minilm|e5-/i.test(m));
      // Offered only while it isn't a source yet.
      $("ollama-box").hidden = !(o.available && chat.length) || S.providers.some((p) => p.kind === "ollama");
      $("ollama-models").textContent = chat.join(", ");
    } catch (_) { /* no local model */ }
  } else if (S.devices.length === 0 && !store("omnillm.skipPair")) {
    stepTo(2);
    // A device added: that is the goal; its first message is sent there.
    startPair($("pair-area"), () => { store("omnillm.wizard", "done"); toast(t("pairedTry")); refresh(); }, false);
  } else {
    stepTo(3);
    fillModels();
  }
}

function fillModels() {
  const sel = $("try-model");
  sel.textContent = "";
  for (const m of S.models) {
    const o = el("option", m.locality === "local" ? withDot(modelName(m), t("local")) : isSub(m) ? withDot(modelName(m), t("sub")) : modelName(m));
    o.value = m.id;
    sel.append(o);
  }
}

noImeSubmit($("key-form"));
noImeSubmit($("try-form"));
$("kind").addEventListener("change", () => { $("compat").hidden = $("kind").value !== "openai-compatible"; });

$("key-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const btn = ev.submitter || $("key-form").querySelector("button");
  btn.disabled = true;
  const body = { kind: $("kind").value, key: $("key").value.trim() };
  if (body.kind === "openai-compatible") { body.name = $("cname").value.trim(); body.base_url = $("curl").value.trim(); }
  try {
    const r = await api("providers/add", body);
    $("key").value = "";
    toast(connectedText(r.models.length));
    await refresh();
  } catch (e) { toast(e.message, true); } finally { btn.disabled = false; }
});

// connectedText: "Connected: 3 models available" (one: "1 model").
const connectedText = (n) => n === 1 ? t("connectedOne") : t("connected", n);

$("ollama-use").addEventListener("click", async (ev) => {
  ev.target.disabled = true;
  try {
    const r = await api("providers/add", { kind: "ollama", key: "" });
    toast(connectedText(r.models.length));
    await refresh();
  } catch (e) { toast(e.message, true); } finally { ev.target.disabled = false; }
});

// "Check again": the hub connects every subscription signed in on this Mac
// (even one removed before: asking is adding it again) and a running
// Ollama, and says which by their names, then what it found but could not
// connect, each with why.
$("recheck").addEventListener("click", async (ev) => {
  const b = ev.currentTarget;
  b.disabled = true;
  try {
    const r = await api("providers/detect", {});
    const kinds = r.connected || [], all = r.failed || [];
    // Notes (an Ollama without a chat model) are said, never in red, and
    // not at all when a model on this Mac is already connected.
    const haveLocal = S.providers.some((p) => p.local && !p.off);
    const failed = all.filter((f) => !f.note), notes = haveLocal ? [] : all.filter((f) => f.note);
    const name = (k) => sourceName(k) || k;
    const lines = [];
    if (kinds.length) lines.push(t("connectedNames", kinds.map(name).join(lang === "zh" ? "、" : ", ")));
    else if (!failed.length && !notes.length) lines.push(S.providers.length ? t("recheckNoMore") : t("recheckNone"));
    for (const f of failed) lines.push(t("srcFailed", name(f.kind), f.message));
    for (const f of notes) lines.push(f.message);
    toast(lines.join(lang === "zh" ? "" : " "), failed.length > 0, undefined, notes.length > 0); // a note with a command to read
    if (kinds.length) await refresh();
  } catch (e) { toast(e.message, true); } finally { b.disabled = false; }
});

$("skip-pair").addEventListener("click", () => { store("omnillm.skipPair", "1"); closePair($("pair-area")); stepTo(3); fillModels(); });

$("try-form").addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const btn = ev.submitter || $("try-form").querySelector("button");
  btn.disabled = true;
  const out = $("try-reply");
  out.hidden = false;
  out.textContent = "…";
  try {
    const r = await api("test", { model: $("try-model").value, text: $("try-text").value });
    out.textContent = r.reply;
  } catch (e) { out.textContent = e.message; } finally { btn.disabled = false; }
});

$("finish").addEventListener("click", () => { store("omnillm.wizard", "done"); refresh(); });

// ---------- pairing ----------

let pairTimer = null;
function stopPair() { if (pairTimer) { clearInterval(pairTimer); pairTimer = null; } }

// The pairing on screen, so closing the panel can end it at once. pairGen
// counts starts and closes, so a pairing whose start answers after the
// panel was closed (or started again) is dropped instead of shown.
let curPair = null, pairGen = 0;
// pairRedraw draws the open panel again in a new language; null when none is open.
let pairRedraw = null;
function cancelPair() { if (curPair) { api("pair/cancel", { id: curPair }).catch(() => {}); curPair = null; } }
function closePair(area) {
  pairGen++;
  pairRedraw = null;
  stopPair();
  cancelPair();
  $("add-device").hidden = false;
  area.textContent = "";
  area.classList.remove("pair-panel");
}

// startPair shows "add device" in area. Two ways, one at a time: a phone
// scans the QR code; a computer opens the same link in its browser (the
// command line is folded away for one without a browser). The left side
// changes in place (code or link, then the 6 digits); the right side is
// three steps, a live status line and, at the end, the owner's yes or no.
// closable: the dashboard's panel with its title and x. onDone gets the
// new device's id.
async function startPair(area, onDone, closable) {
  const gen = ++pairGen;
  pairRedraw = null;
  stopPair();
  cancelPair();
  if (closable) {
    $("add-device").hidden = true;
    area.classList.add("pair-panel");
  }
  // fresh empties the area but keeps the panel's x.
  const fresh = () => {
    area.textContent = "";
    if (!closable) return;
    const x = el("button", "", "close-x"); x.type = "button"; x.setAttribute("aria-label", t("close")); x.title = t("close");
    x.addEventListener("click", () => closePair(area));
    area.append(x);
  };
  const again = () => startPair(area, onDone, closable);
  // startFailed: no code could be made. Say why and offer to try again.
  const startFailed = (msg) => {
    stopPair();
    fresh();
    const b = el("button", t("retry"), "primary");
    b.type = "button";
    b.addEventListener("click", again);
    area.append(el("p", msg, "err-text"), b);
  };
  fresh();
  if (!S.relay_configured) {
    area.append(el("p", t("noRelay"), "muted"));
    return;
  }
  if (!S.relay_up) {
    // Configured but not connected yet (just started, or the network is
    // down): wait for it here and carry on by itself.
    const w = el("p", t("relayWait"), "muted");
    area.append(w);
    const since = Date.now();
    pairTimer = setInterval(async () => {
      let r;
      try { r = await api("relay"); } catch (_) { return; }
      if (gen !== pairGen) return;
      S.relay_up = r.relay_up; S.relay_configured = r.relay_configured;
      if (r.relay_up || !r.relay_configured) { renderStatus(); again(); return; }
      if (Date.now() - since > 15000) w.textContent = t("relayDown");
    }, 1000);
    return;
  }
  let p;
  try { p = await api("pair/start", {}); } catch (e) {
    if (gen !== pairGen || e.status === 401) return;
    // The hub's own sentence for a missing relay promises a reconnect that
    // this button is the way to; say plainly why there is no code.
    startFailed(e.status === undefined ? t("hubDown") : e.code === "hub_offline" ? t("pairStartOffline") : e.message);
    return;
  }
  if (gen !== pairGen) { api("pair/cancel", { id: p.id }).catch(() => {}); return; }
  curPair = p.id;
  const started = Date.now(), expires = new Date(p.expires).getTime();
  // A link another computer can open; a hub whose web page is its own
  // address (localhost) has none, and computers use the command line.
  const link = /^https?:\/\/[^\s]+#/.test(p.link || "") && !loopback(p.link) ? p.link : "";
  // phase: wait (for the device), confirm (the owner's yes or no), ended
  // (expired, refused, or lost: a new code is one click away).
  let phase = "wait", ask = null, endKind = "", endMsg = "", endErr = false, cliOpen = false, nameVal = null;
  let clock = null, arc = null;
  const way = () => (pairTab === "phone" ? "phone" : link ? "link" : "cli");
  const tick = () => {
    if (!clock) return;
    const left = Math.max(0, expires - Date.now());
    const s = Math.ceil(left / 1000);
    clock.textContent = Math.floor(s / 60) + ":" + String(s % 60).padStart(2, "0");
    arc.setAttribute("stroke-dashoffset", String(RING * (1 - left / Math.max(1, expires - started))));
  };
  const draw = () => {
    fresh();
    clock = arc = null;
    if (closable) area.append(el("h2", t("addDevice"), "pair-title"));
    if (phase !== "confirm") area.append(pairTabs(() => draw()));
    const body = el("div", undefined, "pair");
    const hero = el("div", undefined, "pair-hero");
    const side = el("div", undefined, "pair-side");
    body.append(hero, side);
    area.append(body);
    const w = way();
    if (phase === "confirm") {
      const d = el("div", undefined, "pair-card pair-digits");
      d.append(el("span", t("pairHere"), "muted small"), el("div", String(ask.code).replace(/^(\d{3})(\d{3})$/, "$1 $2"), "sas"), el("span", ask.device_name, "pair-dev"));
      hero.append(d);
    } else {
      const card = el("div", undefined, "pair-card");
      if (w === "phone") {
        card.classList.add("pair-qr");
        const img = el("img");
        img.alt = t("qrAlt");
        img.src = p.qr;
        card.append(img);
        for (const c of ["tl", "tr", "bl", "br"]) card.append(el("i", undefined, "cor " + c));
      } else if (w === "link") {
        card.classList.add("pair-link");
        const btn = el("button", t("copyLink"), "primary");
        btn.type = "button";
        btn.addEventListener("click", async () => {
          if (!(await copyText(link))) { btn.textContent = t("copyFailed"); return; }
          btn.textContent = t("linkCopied");
          setTimeout(() => { btn.textContent = t("copyLink"); }, 2500);
        });
        card.append(icon(ICONS.web, "pair-ic"), el("span", link.replace(/^https?:\/\//, "").split("#")[0].replace(/\/$/, "") + "/#pair=…", "pair-url"), btn);
      } else {
        card.classList.add("pair-link");
        card.append(icon(ICONS.term, "pair-ic"), el("span", t("pairCliHero"), "pair-url"));
      }
      if (phase === "ended") {
        card.classList.add("ended");
        const v = el("div", undefined, "pair-veil");
        const b = el("button", t(w === "phone" ? "pairNew" : w === "link" ? "pairNewLink" : "pairNewText"), "primary");
        b.type = "button";
        b.addEventListener("click", again);
        v.append(el("b", endKind === "expired" ? t(w === "phone" ? "pairExpired" : w === "link" ? "pairExpiredLink" : "pairExpiredText") : t("pairEnded")), b);
        card.append(v);
      }
      hero.append(card);
      if (phase === "wait") {
        const row = el("div", undefined, "pair-timer");
        const r = ring();
        arc = r.arc;
        clock = el("b");
        const [a, b] = t("pairExpiresIn", "\u0000").split("\u0000");
        const txt = el("span");
        txt.append(a, clock, b);
        row.append(r.svg, txt);
        const n = el("button", t(w === "phone" ? "pairNew" : w === "link" ? "pairNewLink" : "pairNewText"), "link");
        n.type = "button";
        n.addEventListener("click", again);
        hero.append(row, n);
        tick();
      }
    }
    const steps = { phone: ["pairP1", "pairP2"], link: ["pairL1", "pairL2"], cli: ["pairK1", "pairK2"] }[w].concat("pairS3");
    const cur = phase === "confirm" ? 2 : 0;
    const ol = el("ol", undefined, "pair-steps");
    steps.forEach((k, i) => {
      const li = el("li", undefined, i < cur ? "ok" : i === cur ? "now" : "");
      const n = el("span", i < cur ? "" : String(i + 1), "n");
      if (i < cur) n.append(icon(ICONS.check));
      const tx = el("div");
      tx.append(el("h3", t(k)), el("p", t(k + "d")));
      li.append(n, tx);
      ol.append(li);
    });
    side.append(ol);
    const status = el("div", undefined, "pair-status" + (phase === "ended" && endErr ? " err" : ""));
    if (phase !== "ended") status.append(el("span", undefined, "pulse"));
    status.append(el("span", phase === "confirm" ? t("pairConfirmQ", ask.device_name)
      : phase === "ended" ? endMsg : t({ phone: "pairWaitPhone", link: "pairWaitLink", cli: "pairWaitCli" }[w])));
    side.append(status);
    if (phase === "confirm") {
      // The owner may name the device while confirming.
      const nid = "pair-name-" + p.id.slice(0, 8);
      const nl = el("label", t("name")); nl.htmlFor = nid;
      const nameIn = el("input"); nameIn.id = nid; nameIn.maxLength = 64; nameIn.value = nameVal ?? (ask.device_name || "");
      nameIn.addEventListener("input", () => { nameVal = nameIn.value; });
      const actions = el("div", undefined, "actions");
      const yes = el("button", t("pairYes"), "primary");
      const no = el("button", t("pairNo"), "secondary");
      yes.type = no.type = "button";
      const decide = async (v) => {
        yes.disabled = no.disabled = true;
        try { await api("pair/decide", { id: p.id, yes: v, name: v ? nameIn.value.trim() : "" }); } catch (e) { yes.disabled = no.disabled = false; toast(e.message, true); }
      };
      yes.addEventListener("click", () => decide(true));
      no.addEventListener("click", () => decide(false));
      nameIn.addEventListener("keydown", (e) => { if (e.key === "Enter" && !composing(e) && !yes.disabled) decide(true); });
      actions.append(no, yes);
      side.append(el("div", undefined, "pair-name"), actions);
      side.querySelector(".pair-name").append(nl, nameIn);
      // On a phone the digits sit above the question: bring them into view.
      if (matchMedia("(max-width: 600px)").matches) hero.scrollIntoView({ block: "start" }); else area.scrollIntoView({ block: "nearest" });
      yes.focus({ preventScroll: true });
    }
    if (phase === "wait" && w !== "phone") {
      // The command line: folded behind one quiet line when the browser
      // link is there, open when it is the only way.
      const cli = el("div", undefined, "pair-cli" + (cliOpen || !link ? " open" : ""));
      if (link) {
        const tg = el("button", t("pairCliToggle"), "link");
        tg.type = "button";
        tg.setAttribute("aria-expanded", String(cliOpen));
        tg.append(el("span", "›", "chev"));
        tg.addEventListener("click", () => { cliOpen = !cliOpen; draw(); });
        cli.append(tg);
      }
      if (cliOpen || !link) {
        const ol2 = el("ol");
        const item = (k, shown, full) => {
          const li = el("li");
          li.append(el("b", t(k)), el("span", t(k + "d"), "hint"), copyRow(shown, full));
          ol2.append(li);
        };
        item("pairCmd1", "curl -fsSL …/install.sh | sh -s -- --device", PAIR_INSTALL);
        item("pairCmd2", p.code.slice(0, 18) + " … " + p.code.slice(-8), p.code);
        cli.append(ol2);
      }
      side.append(cli);
    }
  };
  const end = (kind, msg, isErr) => {
    stopPair();
    cancelPair();
    phase = "ended"; endKind = kind; endMsg = msg; endErr = !!isErr;
    draw();
  };
  draw();
  pairRedraw = () => { if (gen === pairGen) draw(); };
  pairTimer = setInterval(async () => {
    tick();
    let st;
    try { st = await api("pair/state?id=" + encodeURIComponent(p.id)); } catch (e) {
      if (gen !== pairGen) return;
      if (e.status === 404) { end("expired", t("pairExpiredSay")); return; } // gone on the hub: expired
      if (e.status === undefined) return; // the hub isn't answering: the banner says so; keep asking
      if (e.status !== 401) end("failed", e.message, true);
      return;
    }
    // Two polls may overlap: only the first one past the end acts.
    if (gen !== pairGen || phase === "ended" || !pairTimer) return;
    if (st.state === "waiting" && Date.now() > expires) { end("expired", t("pairExpiredSay")); return; }
    if (st.state === "confirm" && phase === "wait") {
      phase = "confirm";
      ask = st;
      draw();
    }
    if (st.state === "done") {
      pairGen++; // a slower poll still in flight finds this pairing over
      stopPair();
      curPair = null;
      pairRedraw = null;
      area.textContent = "";
      area.append(el("p", t("pairDone", st.device_name)));
      toast(named("pairDone", st.device_name));
      onDone && onDone(st.device_id);
    }
    if (st.state === "failed") {
      curPair = null;
      if (st.reason === "denied") end("denied", t("pairDenied"));
      else if (st.reason === "left") end("left", t("pairLeft"));
      else if (st.reason === "relay") end("relay", t("pairRelayDrop"), true);
      else if (st.reason === "expired") end("expired", t("pairExpiredSay"));
      else end("failed", t("pairFailed", st.message || ""), true);
    }
  }, 1000);
}

// pairTab: the way "add device" shows, phone (QR code) or computer (a link
// for its browser); kept while the page is open.
let pairTab = "phone";
const PAIR_INSTALL = "curl -fsSL https://raw.githubusercontent.com/wensuper/OmniLLM/main/install.sh | sh -s -- --device";

// pairTabs: the phone | computer switch above the panel.
function pairTabs(redraw) {
  const seg = el("div", undefined, "pair-seg");
  seg.setAttribute("role", "tablist");
  for (const [k, key, ic] of [["phone", "pairTabPhone", ICONS.phone], ["computer", "pairTabComputer", ICONS.computer]]) {
    const b = el("button");
    b.type = "button";
    b.setAttribute("role", "tab");
    b.setAttribute("aria-selected", String(pairTab === k));
    b.append(icon(ic), t(key));
    b.addEventListener("click", () => { if (pairTab !== k) { pairTab = k; redraw(); } });
    seg.append(b);
  }
  return seg;
}

// copyText puts text on the clipboard, falling back to the old copy
// command where the Clipboard API is missing; false when neither works.
async function copyText(text) {
  try { await navigator.clipboard.writeText(text); return true; } catch (_) { /* below */ }
  const ta = el("textarea");
  ta.value = text;
  ta.setAttribute("readonly", "");
  ta.className = "offscreen";
  document.body.append(ta);
  ta.select();
  let ok = false;
  try { ok = document.execCommand("copy"); } catch (_) { ok = false; }
  ta.remove();
  return ok;
}

// copyRow: one line shown shortened (the middle left out, both ends kept,
// so "--device" stays in sight), with Copy copying all of it.
function copyRow(shown, full) {
  const r = el("div", undefined, "copyrow");
  const s = el("span", shown);
  s.title = full;
  r.append(s, copyBtn(full));
  return r;
}

// Small line icons, drawn as SVG elements (the page sets no HTML).
const ICONS = {
  phone: [["rect", { x: 7, y: 2.5, width: 10, height: 19, rx: 2.5 }], ["path", { d: "M11 18.5h2" }]],
  computer: [["rect", { x: 3, y: 4, width: 18, height: 12, rx: 2 }], ["path", { d: "M8 20h8M12 16v4" }]],
  web: [["circle", { cx: 12, cy: 12, r: 9 }], ["path", { d: "M3 12h18M12 3c3 3.5 3 14.5 0 18M12 3c-3 3.5-3 14.5 0 18" }]],
  term: [["rect", { x: 3, y: 4, width: 18, height: 16, rx: 2 }], ["path", { d: "M7 9l3 3-3 3M12 15h5" }]],
  check: [["path", { d: "M5 12.5l4.5 4.5L19 7.5" }]],
};
function svg(tag, attrs) {
  const n = document.createElementNS("http://www.w3.org/2000/svg", tag);
  for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, String(v));
  return n;
}
function icon(parts, cls) {
  const s = svg("svg", { viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": 2, "stroke-linecap": "round", "stroke-linejoin": "round", "aria-hidden": "true" });
  if (cls) s.setAttribute("class", cls);
  for (const [tag, a] of parts) s.append(svg(tag, a));
  return s;
}
// ring: the countdown circle; its arc shrinks as the code runs out.
const RING = 2 * Math.PI * 7;
function ring() {
  const s = svg("svg", { viewBox: "0 0 18 18", class: "ring", "aria-hidden": "true" });
  s.append(svg("circle", { cx: 9, cy: 9, r: 7, fill: "none", stroke: "var(--line-2, var(--line))", "stroke-width": 2.4 }));
  const arc = svg("circle", { cx: 9, cy: 9, r: 7, fill: "none", stroke: "var(--accent)", "stroke-width": 2.4, "stroke-linecap": "round", "stroke-dasharray": RING, "stroke-dashoffset": 0, transform: "rotate(-90 9 9)" });
  s.append(arc);
  return { svg: s, arc };
}

// loopback: the address names this computer itself (localhost, 127.x,
// ::1), which only a browser here can open.
function loopback(web) {
  let h;
  try { h = new URL(web).hostname.toLowerCase(); } catch (_) { return false; }
  return h === "localhost" || h.endsWith(".localhost") || /^127\.\d+\.\d+\.\d+$/.test(h) || h === "0.0.0.0" || h === "[::1]" || h === "::1";
}

// ---------- dashboard ----------

function showDash() {
  closeCard(); // its target is about to be drawn again
  show("dash");
  fromDash = false;
  const total = (S.usage || []).find((u) => u.who === "_total");
  $("spend-today").textContent = money(total ? total.micro_usd : 0);
  renderCap();
  renderDevices();
  renderModels();
  renderApps();
  renderUsage();
}

// isSub: a Claude or ChatGPT subscription (the official programs on this
// computer), not merely a model kept for people.
function isSub(m) { const k = kindOf(m.provider); return k === "claude-code" || k === "codex" || k === "antigravity"; }
// kindOf: what kind of source a provider name is (its kind, as connected).
function kindOf(name) { const p = S.providers.find((x) => x.name === name); return p ? p.kind : name; }
function cell(row, text, cls) { const td = el("td", text, cls); row.append(td); return td; }

// removeButton: Remove in a table row; it asks in a line of its own right
// under the row (the question, then Cancel and Remove).
function removeButton(label, path, id, redraw, confirmKey = "removeConfirm") {
  const b = el("button", t("remove"), "danger");
  b.type = "button";
  b.addEventListener("click", () => {
    const tr = b.closest("tr");
    tr.classList.add("asking");
    b.hidden = true;
    const ask = el("tr", undefined, "asking ask-row");
    const td = el("td"); td.colSpan = tr.cells.length;
    const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
    const yes = el("button", t("remove"), "danger"); yes.type = "button";
    no.addEventListener("click", redraw);
    yes.addEventListener("click", async () => {
      no.disabled = yes.disabled = true;
      try { await api(path, { id }); toast(t("removed")); refresh(); } catch (e) { toast(e.message, true); no.disabled = yes.disabled = false; }
    });
    const btns = el("div", undefined, "btns"); btns.append(no, yes);
    const line = el("div", undefined, "row-ask"); line.append(el("p", t(confirmKey, label)), btns);
    td.append(line); ask.append(td);
    tr.after(ask);
    no.focus();
  });
  return b;
}

const NO_LIMIT = -1;
// spentText: what a device or app spent today against its own daily limit,
// so a limit is never read without how close it is.
function spentText(who, limit) {
  const u = S.usage.find((x) => x.who === who);
  const today = money(u ? u.micro_usd : 0);
  return limit === NO_LIMIT ? t("todayNoLimit", today) : t("todayOf", today, money(limit));
}

// timeWords: when something last happened, as people say it: today or
// yesterday with the time, else the date.
function timeWords(iso) {
  if (!iso || iso.startsWith("0001")) return t("never");
  const d = new Date(iso), now = new Date();
  const loc = lang === "zh" ? "zh-CN" : "en-US";
  const time = d.toLocaleTimeString(loc, lang === "zh" ? { hour: "2-digit", minute: "2-digit", hour12: false } : { hour: "numeric", minute: "2-digit" });
  const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diff = Math.round((day(now) - day(d)) / 86400000);
  if (diff === 0) return t("todayAt", time);
  if (diff === 1) return t("yesterdayAt", time);
  return dateWords(iso);
}
// inLine: timeWords inside a sentence: "Added today 9:41 AM", not "Today".
const inLine = (s) => lang === "en" ? s.replace(/^(Today|Yesterday) /, (w) => w.toLowerCase()) : s;
// dateWords: a date alone: "10月18日" / "Oct 18" (with the year when it
// is not this year's).
function dateWords(iso) {
  if (!iso || iso.startsWith("0001")) return t("never");
  const d = new Date(iso), other = d.getFullYear() !== new Date().getFullYear();
  if (lang === "zh") return (other ? d.getFullYear() + "年" : "") + (d.getMonth() + 1) + "月" + d.getDate() + "日";
  return d.toLocaleDateString("en-US", other ? { year: "numeric", month: "short", day: "numeric" } : { month: "short", day: "numeric" });
}
// seenText: "Online" while the device has a session open, else when it was
// last used.
function seenText(td, online, iso) {
  if (online) td.append(el("span", "●", "dot-ok"), document.createTextNode(" " + t("online")));
  else td.textContent = timeWords(iso);
}

// ---- hover cards ----
// hoverCard shows build()'s card next to target while the pointer rests on
// it or it has keyboard focus: below it, or above when there is no room;
// Escape, scrolling or leaving closes it. One card at a time.
let hcOpen = null, hcTimer = 0;
function hoverCard(target, build) {
  target.tabIndex = 0;
  target.classList.add("hc-target");
  const open = () => {
    clearTimeout(hcTimer);
    hcTimer = setTimeout(() => {
      closeCard();
      const card = el("div", undefined, "hovercard");
      card.setAttribute("role", "tooltip");
      card.id = "hovercard";
      card.append(build());
      document.body.append(card);
      const r = target.getBoundingClientRect(), c = card.getBoundingClientRect();
      const left = Math.max(8, Math.min(r.left, innerWidth - c.width - 8));
      const below = r.bottom + 8 + c.height <= innerHeight - 8;
      card.style.left = left + scrollX + "px";
      card.style.top = (below ? r.bottom + 8 : r.top - 8 - c.height) + scrollY + "px";
      card.classList.add(below ? "below" : "above");
      target.setAttribute("aria-describedby", "hovercard");
      hcOpen = { card, target };
    }, 180);
  };
  const close = () => { clearTimeout(hcTimer); if (hcOpen && hcOpen.target === target) closeCard(); };
  target.addEventListener("mouseenter", open);
  target.addEventListener("focus", open);
  target.addEventListener("mouseleave", close);
  target.addEventListener("blur", close);
}
function closeCard() {
  if (!hcOpen) return;
  hcOpen.card.remove();
  hcOpen.target.removeAttribute("aria-describedby");
  hcOpen.target.setAttribute("aria-expanded", "false");
  hcOpen = null;
}

// clickCard: a card that opens on click and stays (its text can be
// selected and copied); a second click, a click elsewhere, Escape or
// scrolling closes it.
function clickCard(target, build) {
  target.addEventListener("click", (e) => {
    e.stopPropagation();
    if (hcOpen && hcOpen.target === target) return closeCard();
    closeCard();
    const card = el("div", undefined, "hovercard sticky");
    card.setAttribute("role", "dialog");
    card.setAttribute("aria-label", target.getAttribute("aria-label") || "");
    card.append(build());
    card.addEventListener("click", (ev) => ev.stopPropagation());
    document.body.append(card);
    const r = target.getBoundingClientRect(), c = card.getBoundingClientRect();
    const left = Math.max(8, Math.min(r.left - 12, innerWidth - c.width - 8));
    const below = r.bottom + 8 + c.height <= innerHeight - 8;
    card.style.left = left + scrollX + "px";
    card.style.top = (below ? r.bottom + 8 : r.top - 8 - c.height) + scrollY + "px";
    card.classList.add(below ? "below" : "above");
    target.setAttribute("aria-expanded", "true");
    hcOpen = { card, target };
  });
}
addEventListener("click", () => { if (hcOpen && hcOpen.card.classList.contains("sticky")) closeCard(); });

// appsHelpCard: how a program connects, with the address to copy.
function appsHelpCard() {
  const box = el("div");
  box.append(el("div", t("appsHelp"), "hc-title hc-head"));
  const addr = el("span", undefined, "hc-addr");
  addr.append(el("code", S.api_url), copyBtn(S.api_url));
  box.append(hcRows([[t("apAddr"), addr], [t("apKey"), t("apKeyV")], [t("apFmt"), t("apFmtV")], [t("apWho"), t("apWhoV")]]));
  return box;
}
addEventListener("keydown", (e) => { if (e.key === "Escape") closeCard(); });
addEventListener("scroll", closeCard, true);

// hcRows: label / value lines of a card.
function hcRows(rows) {
  const dl = el("dl", undefined, "hc-rows");
  for (const [k, v] of rows) {
    if (v === undefined || v === null || v === "") continue;
    const dd = v instanceof Node ? el("dd") : el("dd", v);
    if (v instanceof Node) dd.append(v);
    dl.append(el("dt", k), dd);
  }
  return dl;
}

// nameCard: a device's or app's details: what it is, since when, last
// seen, today's spend against its limit.
function nameCard(x, kind, spent) {
  const box = el("div");
  const head = el("div", undefined, "hc-head");
  head.append(el("span", x.Name, "hc-title"));
  if (x.Paused) head.append(el("span", t("pausedChip"), "tag paused-tag"));
  else if (kind === "device" && x.Online) { const on = el("span", t("online"), "hc-on"); on.prepend(el("span", "", "dot-on")); head.append(on); }
  box.append(head);
  // What it is, where that is known (a phone or a computer isn't).
  const what = kind === "app" ? t("hcApp") : x.BackgroundOnly ? t("hcBackground") : "";
  box.append(hcRows([
    [t("hcKind"), what],
    [t("hcAdded"), x.Created ? inLine(timeWords(x.Created)) : ""],
    [t("hcSeen"), kind === "device" && !x.Online ? inLine(timeWords(x.LastSeen)) : ""], // online: the chip says it
    [t("hcToday"), spent],
  ]));
  return box;
}

// modelsCard: which models it can use, by source, the ones not switched
// on shown as off; "all" says later ones come too.
function modelsCard(x, all, on, isAll) {
  const box = el("div");
  box.append(el("div", isAll ? t("hcAllTitle", num(all.length)) : t("hcSomeTitle", num(on.length), num(all.length)), "hc-title"));
  if (!all.length) { box.append(el("p", t("hcNoneUsable"), "muted small")); return box; }
  const ids = new Set(on.map((m) => m.id));
  for (const [g, label] of [["sub", t("grpSub")], ["key", t("grpKey")], ["local", t("grpLocal")]]) {
    const ms = all.filter((m) => (m.group || "key") === g);
    if (!ms.length) continue;
    box.append(el("div", label, "hc-group"));
    const ul = el("ul", undefined, "hc-models");
    for (const m of ms) {
      const li = el("li", undefined, ids.has(m.id) ? "on" : "off");
      li.append(el("span", "", "hc-mark"), el("span", modelName(m)));
      if (!ids.has(m.id)) li.append(el("span", t("hcOff"), "hc-offtag"));
      ul.append(li);
    }
    box.append(ul);
  }
  if (isAll) box.append(el("p", t("hcAllNote"), "muted small hc-note"));
  return box;
}

// nameCell: a device's or app's name, with a "Paused" chip when it is.
// On a phone-width window the row stacks: the name, and under it what it
// spent today against its limit, after a green dot and "Online ·" while
// it is online, or when it was last seen (seen, a device's) while not.
function nameCell(tr, name, paused, online, spent, seen) {
  const td = cell(tr, "", "nm");
  const line = el("div", undefined, "nm-line");
  const nm = el("span", name, "nm-text");
  line.append(nm);
  if (paused) line.append(el("span", t("pausedChip"), "tag paused-tag"));
  td.append(line);
  if (spent) {
    const sub = el("div", undefined, "nm-sub narrow-only parts");
    // Each part stays whole; the "·" is drawn between parts on one line
    // (CSS), so a wrapped line never starts or ends with it.
    const part = (s) => el("span", s, "nowrap part");
    if (online) { const on = part(t("online")); on.prepend(el("span", "", "dot-on")); sub.append(on, part(spent)); }
    else if (seen !== undefined) sub.append(part(timeWords(seen)), part(spent));
    else sub.append(part(spent));
    td.append(sub);
  }
  return td;
}

// modelsCell: which models a device or app can use: all of them, or some
// ("3 / 7": on its list, out of all it could use). A list that holds every
// usable model is all too. The names are in the tooltip; on a phone-width
// window the same words join the line under the name.
function modelsCell(tr, nc, x, usable) {
  const all = S.models.filter(usable);
  const list = x.Models || [];
  const on = list.length ? all.filter((m) => list.includes(m.id)) : all;
  const td = cell(tr, "", "mdl");
  if (!on.length) { td.append(el("span", "—", "muted")); return; }
  const some = on.length < all.length;
  const txt = some ? t("modelsSome", num(on.length), num(all.length)) : t("modelsAll", num(all.length));
  const label = el("span", txt, some ? "some-models" : "");
  td.append(label);
  hoverCard(label, () => modelsCard(x, all, on, list.length === 0 || !some));
  const sub = nc.querySelector(".nm-sub");
  if (sub) sub.append(el("span", txt, "nowrap part"));
}

// pauseSwitch: on = in use; turning it off pauses that device or app at
// once (its next request is refused), and on again resumes it.
function pauseSwitch(tr, label, path, id, paused, field = "paused", toasts = ["pausedToast", "resumedToast"]) {
  const td = cell(tr, "", "sw");
  const sw = el("input"); sw.type = "checkbox"; sw.className = "switch"; sw.checked = !paused;
  sw.setAttribute("aria-label", t("enabled") + " " + label);
  sw.addEventListener("change", async () => {
    sw.disabled = true;
    // Only a failed change puts the switch back; the redraw after it is
    // just a redraw (the next poll tries again).
    try {
      await api(path, { id, [field]: !sw.checked });
    } catch (e) { sw.checked = !sw.checked; sw.disabled = false; toast(e.message, true); return; }
    if (sw.checked) toast(named(toasts[1], label));
    else {
      // Paused by a slip of the finger: Undo turns it on again.
      toast(named(toasts[0], label), false, { label: t("undo"), run: async () => {
        try { await api(path, { id, [field]: false }); toast(named(toasts[1], label)); await refresh(); } catch (e) { toast(e.message, true); }
      } });
    }
    try { await refresh(); } catch (_) { sw.disabled = false; }
  });
  td.append(sw);
}

// fresh: the device just added, lit up (--accent-soft) until its time is
// up, through the redraws that follow while it comes online.
let fresh = { id: "", until: 0 };
function renderDevices() {
  const tb = $("devices");
  tb.textContent = "";
  const n = S.devices.length;
  $("devices-count").textContent = n ? (n === 1 ? t("deviceCountOne") : t("deviceCount", num(n))) : "";
  for (const d of S.devices) {
    const tr = el("tr", undefined, d.Paused ? "paused" : "");
    tr.dataset.id = d.ID;
    if (d.ID === fresh.id && Date.now() < fresh.until) tr.classList.add("fresh");
    const spent = spentText("device/" + d.ID, d.BudgetMicroUSD);
    const nc = nameCell(tr, d.Name, d.Paused, d.Online, spent, d.LastSeen);
    hoverCard(nc.querySelector(".nm-text"), () => nameCard(d, "device", spent));
    // Nothing on its list is left for it (removed, or kept off devices).
    if (!deviceHasModels(d)) {
      const tag = el("span", t("appNoModels"), "tag warn-tag");
      tag.title = t(devNoModelsWhy(d));
      nc.firstChild.append(el("span", "", "brk narrow-only"), tag);
    }
    seenText(cell(tr, "", "seen"), d.Online, d.LastSeen);
    cell(tr, spent, "spent");
    modelsCell(tr, nc, d, (m) => usableByDevice(d, m));
    pauseSwitch(tr, d.Name, "devices/update", d.ID, d.Paused);
    const ed = el("button", t("edit"), "secondary small-btn");
    ed.type = "button";
    ed.addEventListener("click", () => editDialog("device", d));
    cell(tr, "", "actions").append(ed);
    tb.append(tr);
  }
  $("devices-empty").hidden = S.devices.length > 0;
}

// moneyInput: a dollar amount without the browser's number spinners; the
// "$" sits inside the field, before the amount.
function moneyInput(value, label) {
  const wrap = el("span", undefined, "money-in");
  const amt = el("input"); amt.type = "text"; amt.inputMode = "decimal"; amt.autocomplete = "off"; amt.spellcheck = false;
  amt.value = value; amt.setAttribute("aria-label", label);
  const cur = el("span", "$", "money-sign"); cur.setAttribute("aria-hidden", "true");
  wrap.append(cur, amt);
  return [wrap, amt];
}
// amount: what a money field starts with: "5.00".
const amount = (micro) => (micro / 1e6).toFixed(2);
// dollars reads an amount; anything unreadable is sent as 0, which the
// hub refuses with its own sentence.
function dollars(v) { const n = Number(String(v).trim().replace(",", ".")); return Number.isFinite(n) ? n : 0; }

// renderCap: the whole hub's daily total next to today's spend, changed in
// place (an amount, or no limit).
function renderCap() {
  const box = $("cap"); box.textContent = "";
  const v = S.daily_total_micro_usd;
  box.append(el("span", v === NO_LIMIT ? t("capNone") : t("capLine", money(v)), "muted"));
  const b = el("button", t("capEdit"), "link"); b.type = "button";
  b.addEventListener("click", () => {
    box.textContent = "";
    const f = el("div", undefined, "cap-edit");
    const [wrap, amt] = moneyInput(amount(v === NO_LIMIT ? 20e6 : v), t("capLabel"));
    amt.disabled = v === NO_LIMIT;
    const sw = el("input"); sw.type = "checkbox"; sw.className = "switch"; sw.checked = v === NO_LIMIT;
    const lab = el("label", undefined, "toggle"); lab.append(sw, el("span", t("noLimit")));
    sw.addEventListener("change", () => { amt.disabled = sw.checked; });
    const save = el("button", t("save"), "primary small-btn"); save.type = "button";
    const no = el("button", t("cancel"), "secondary small-btn"); no.type = "button";
    const submit = async () => {
      save.disabled = true;
      try { await api("daily-total", sw.checked ? { no_limit: true } : { budget_usd: dollars(amt.value) }); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); save.disabled = false; }
    };
    save.addEventListener("click", submit);
    no.addEventListener("click", renderCap);
    // Enter in the amount saves; on a button, the button does its own thing.
    f.addEventListener("keydown", (e) => { if (composing(e)) return; if (e.key === "Enter" && e.target === amt) submit(); if (e.key === "Escape") renderCap(); });
    const fields = el("div", undefined, "cap-fields");
    fields.append(el("span", t("capLabel"), "cap-label"), wrap, lab);
    const btns = el("div", undefined, "btns");
    btns.append(no, save);
    f.append(fields, btns);
    box.append(f);
    amt.focus();
  });
  box.append(b);
}

// ---------- edit dialogs ----------

// dialog: the frame every edit dialog shares: a head with its title and
// x, a body, and a foot with Cancel and then Save on the right (left: an
// optional button such as Remove). With nothing changed, Esc, the x and
// the backdrop close it; with unsaved changes the backdrop does nothing,
// and Esc or the x ask first, in the foot. Enter in a field saves when
// Save can. Tab stays inside. Questions are asked in the foot, never in a
// browser window.
function dialog(titleText) {
  const opener = document.activeElement;
  const scrim = el("div", undefined, "dialog-scrim");
  const dlg = el("div", undefined, "dialog");
  dlg.setAttribute("role", "dialog"); dlg.setAttribute("aria-modal", "true"); dlg.tabIndex = -1;
  const head = el("div", undefined, "dialog-head");
  const title = el("h2", titleText); title.id = "dlg-title";
  dlg.setAttribute("aria-labelledby", title.id);
  const xb = el("button", "", "close-x"); xb.type = "button"; xb.setAttribute("aria-label", t("close")); xb.title = t("close");
  head.append(title, xb);
  const body = el("div", undefined, "dialog-body");
  const foot = el("div", undefined, "dialog-foot");
  const no = el("button", t("cancel"), "secondary"); no.type = "button";
  const save = el("button", t("save"), "primary"); save.type = "button";
  const right = el("div", undefined, "btns"); right.append(no, save);
  const d = { dlg, body, save, left: null, asking: false, dirty: () => false };
  const onKey = (e) => {
    if (composing(e)) return;
    // A question opened over this dialog answers its own keys.
    const all = document.querySelectorAll(".dialog-scrim");
    if (all[all.length - 1] !== scrim) return;
    if (e.key === "Escape") { e.preventDefault(); if (d.asking) d.showFoot(); else leave(); return; }
    if (e.key === "Enter" && e.target.tagName === "INPUT" && e.target.type !== "checkbox" && dlg.contains(e.target)) {
      e.preventDefault();
      if (!save.disabled && !d.asking) save.click();
      return;
    }
    if (e.key !== "Tab") return;
    const f = [...dlg.querySelectorAll("button, input, select, textarea, [tabindex]")].filter((n) => !n.disabled && n.offsetParent !== null);
    if (!f.length) return;
    const first = f[0], last = f[f.length - 1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    else if (!dlg.contains(document.activeElement) || document.activeElement === dlg) { e.preventDefault(); first.focus(); }
  };
  d.close = () => { scrim.remove(); document.removeEventListener("keydown", onKey, true); if (opener && opener.isConnected) opener.focus(); };
  // showFoot: the foot as it normally is.
  d.showFoot = () => {
    d.asking = false;
    foot.textContent = ""; foot.classList.remove("asking");
    if (d.left) foot.append(d.left);
    foot.append(right);
  };
  // ask: a question in the foot: Cancel (or back) goes back to the form,
  // the other button does it.
  d.ask = (q, yesLabel, onYes, backLabel) => {
    d.asking = true;
    foot.textContent = ""; foot.classList.add("asking");
    const back = el("button", backLabel || t("cancel"), "secondary"); back.type = "button";
    const yes = el("button", yesLabel, "danger"); yes.type = "button";
    back.addEventListener("click", () => { d.showFoot(); });
    yes.addEventListener("click", async () => { back.disabled = yes.disabled = true; if (await onYes() === false) back.disabled = yes.disabled = false; });
    const btns = el("div", undefined, "btns"); btns.append(back, yes);
    foot.append(el("p", q, "foot-q"), btns);
    back.focus();
  };
  const leave = () => { if (d.dirty()) d.ask(t("discardQ"), t("discard"), d.close, t("keepEditing")); else d.close(); };
  xb.addEventListener("click", leave);
  no.addEventListener("click", d.close);
  document.addEventListener("keydown", onKey, true);
  scrim.addEventListener("mousedown", (e) => { if (e.target === scrim && !d.dirty()) d.close(); });
  dlg.append(head, body, foot);
  scrim.append(dlg);
  // open: shows it. On a wide screen the first field takes the caret, at
  // the end of what is there; on a phone nothing is focused, so the
  // keyboard does not cover the dialog before anything is chosen.
  d.open = (field) => {
    d.showFoot();
    document.body.append(scrim);
    if (field && matchMedia("(min-width: 601px)").matches) { field.focus(); const n = field.value.length; field.setSelectionRange(n, n); }
    else dlg.focus({ preventScroll: true });
  };
  return d;
}

// section: a part of a dialog's body, with a heading when it has one.
function section(body, label) {
  const sec = el("section", undefined, "dlg-sec");
  if (label) sec.append(el("h3", label));
  body.append(sec);
  return sec;
}

// usableByApp: apps make background calls only: no chat-only models.
const usableByApp = (m) => !(m.human_only || m.group === "sub" || m.source_off);
// appHasModels: an app can use at least one connected model.
function appHasModels(a) {
  return S.models.some((m) => usableByApp(m) && (!(a.Models || []).length || a.Models.includes(m.id)));
}
// usableByDevice: a device gets the models shown on devices; one that
// only makes background calls (a server, a script) no chat-only ones.
const usableByDevice = (d, m) => !m.hidden && !m.source_off && (!d.BackgroundOnly || usableByApp(m));
// deviceHasModels: a device can use at least one model: one on its list
// (or any, with all models) that it can use.
// devNoModelsWhy: why a device has nothing to use. A device for programs
// only gets no "chat only" model: that is the usual reason when its
// models are still there.
function devNoModelsWhy(d) {
  const listed = S.models.filter((m) => !m.hidden && (!(d.Models || []).length || d.Models.includes(m.id)));
  return d.BackgroundOnly && listed.length ? "devNoModelsBg" : "devNoModelsBody";
}
function deviceHasModels(d) {
  return S.models.some((m) => usableByDevice(d, m) && (!(d.Models || []).length || d.Models.includes(m.id)));
}

// editDialog: everything about one device or app in one place: its name,
// on or paused, its daily limit, the models it may use, and removing it.
function editDialog(kind, x) {
  const isApp = kind === "app";
  const path = isApp ? "apps/update" : "devices/update";
  const who = (isApp ? "app/" : "device/") + x.ID;
  const d = dialog(t(isApp ? "editApp" : "editDevice", x.Name));
  // The name, and what is known about it.
  const basics = section(d.body);
  const nid = "dlg-name";
  const nl = el("label", t("name")); nl.htmlFor = nid;
  const name = el("input"); name.id = nid; name.maxLength = 64; name.value = x.Name;
  const u = S.usage.find((r) => r.who === who);
  const info = [t("infoAdded", inLine(timeWords(x.Created)))];
  if (!isApp) info.push(x.Online ? t("online") : t("infoSeen", inLine(timeWords(x.LastSeen))));
  info.push(t("infoSpent", money(u ? u.micro_usd : 0)));
  // Each "label value" stays whole; the "·" is drawn between parts on one
  // line (CSS), so a wrapped line never starts or ends with it.
  const meta = el("p", undefined, "muted small dlg-info parts");
  info.forEach((s) => meta.append(el("span", s, "nowrap part")));
  basics.append(nl, name, meta);
  // Status: on, or paused.
  const status = section(d.body, t("secStatus"));
  const on = el("input"); on.type = "checkbox"; on.className = "switch"; on.checked = !x.Paused;
  const onl = el("label", undefined, "toggle"); onl.append(on, el("span", t("enabled")));
  status.append(onl, el("p", t(isApp ? "pauseHintApp" : "pauseHint"), "muted small"));
  // Daily limit.
  const lim = section(d.body, t("budget"));
  const unl = x.BudgetMicroUSD === NO_LIMIT;
  const [wrap, amt] = moneyInput(amount(unl ? 5e6 : x.BudgetMicroUSD), t("budgetLabel"));
  amt.disabled = unl;
  const nolim = el("input"); nolim.type = "checkbox"; nolim.className = "switch"; nolim.checked = unl;
  const nll = el("label", undefined, "toggle"); nll.append(nolim, el("span", t("noLimit")));
  const line = el("div", undefined, "budget-line"); line.append(wrap, nll);
  const lhint = el("p", S.daily_total_micro_usd === NO_LIMIT ? t("limitHintNone") : t("limitHint", money(S.daily_total_micro_usd)), "muted small");
  lim.append(line, lhint);
  // Models it can use: all (also later ones), or the ones switched on. A
  // device never gets a model kept off devices; an app (or a device that
  // makes background calls only) never a chat-only one.
  const msec = section(d.body, t("secModels"));
  const usable = (m) => isApp ? usableByApp(m) : usableByDevice(x, m);
  if (isApp && !S.models.some(usableByApp)) msec.append(el("p", t("appNoModelsBody"), "warn-note"));
  if (!isApp && !deviceHasModels(x)) msec.append(el("p", t(devNoModelsWhy(x)), "warn-note"));
  const allSw = el("input"); allSw.type = "checkbox"; allSw.className = "switch";
  const all = el("label", undefined, "toggle wrap"); all.append(allSw, el("span", t("allModels")));
  const list = el("div", undefined, "model-pick");
  const picks = [];
  for (const [g, label] of [["sub", t("grpSub")], ["key", t("grpKey")], ["local", t("grpLocal")]]) {
    const ms = S.models.filter((m) => (m.group || "key") === g);
    if (!ms.length) continue;
    const grp = el("div", undefined, "pick-group");
    grp.append(el("div", label, "pick-label"));
    for (const m of ms) {
      // The switch first, then the name; in an app's list a chat-only
      // model says so, and why it can't be switched on.
      const row = el("label", undefined, "pick-row");
      const nm = el("span", modelName(m), "pick-name");
      if (isApp && !usableByApp(m)) { nm.append(" ", el("span", t("humanOnly"), "tag")); row.title = t("appNoHuman"); }
      if (!isApp && m.hidden) nm.append(" ", el("span", t("hiddenTag"), "tag paused-tag"));
      if (m.source_off) nm.append(" ", el("span", t("sourceOffTag"), "tag paused-tag"));
      const sw = el("input"); sw.type = "checkbox"; sw.className = "switch";
      sw.disabled = !usable(m);
      sw.checked = usable(m) && (!(x.Models || []).length || x.Models.includes(m.id));
      if (sw.disabled) row.classList.add("off");
      row.append(sw, nm);
      grp.append(row);
      picks.push([m.id, sw, usable(m)]);
    }
    list.append(grp);
  }
  if (isApp && S.models.some((m) => !usableByApp(m))) list.prepend(el("p", t("appNoHuman"), "muted small pick-note"));
  const pickHint = el("p", t("pickOne"), "err-text small");
  msec.append(all, list, pickHint);
  // Foot: Remove on the left; it asks in the foot first.
  const rm = el("button", t(isApp ? "removeApp" : "removeDevice"), "danger-text"); rm.type = "button";
  rm.addEventListener("click", () => d.ask(t(isApp ? "removeConfirm" : "removeConfirmDevice", x.Name), t("remove"), async () => {
    try { await api(isApp ? "apps/remove" : "devices/remove", { id: x.ID }); d.close(); toast(t("removed")); refresh(); } catch (e) { toast(e.message, true); return false; }
  }));
  d.left = rm;
  // What the form says now, and whether that differs from what is saved.
  const chosen = () => picks.filter(([, sw, ok]) => ok && sw.checked).map(([id]) => id);
  // "All models" is on whenever every model it can use is on: saved as
  // all (an empty list, so models added later come too), or every one
  // picked by hand. The models stay switchable: switching one off while
  // "all" is on turns "all" off and keeps the rest on.
  const usablePicks = picks.filter(([, , ok]) => ok);
  const everyOn = () => usablePicks.length > 0 && usablePicks.every(([, sw]) => sw.checked);
  allSw.checked = !(x.Models || []).length || everyOn();
  const showAll = () => { if (allSw.checked) for (const [, sw] of usablePicks) sw.checked = true; };
  showAll();
  const state = () => JSON.stringify([name.value.trim(), on.checked, nolim.checked, nolim.checked ? 0 : dollars(amt.value), allSw.checked ? [] : chosen()]);
  const start = state();
  d.dirty = () => state() !== start;
  const sync = () => {
    amt.disabled = nolim.checked;
    lhint.hidden = !nolim.checked;
    const none = !allSw.checked && chosen().length === 0;
    pickHint.hidden = !none;
    d.save.disabled = none || !name.value.trim() || !d.dirty();
  };
  allSw.addEventListener("change", () => {
    // Turning "all" on switches every model on; off keeps them as they are.
    showAll();
    sync();
  });
  // One switched off is no longer "all"; the last one switched on is.
  for (const [, sw] of usablePicks) sw.addEventListener("change", () => { allSw.checked = everyOn(); });
  d.dlg.addEventListener("input", sync);
  d.dlg.addEventListener("change", sync);
  // Models on its list that can't be switched here (kept off devices, or
  // chat-only for an app) stay on it: the owner didn't switch them off.
  const kept = (x.Models || []).filter((id) => !picks.some(([pid, , ok]) => pid === id && ok));
  d.save.addEventListener("click", async () => {
    const b = { id: x.ID, name: name.value, paused: !on.checked, models: allSw.checked ? [] : [...chosen(), ...kept] };
    if (nolim.checked) b.no_limit = true; else b.budget_usd = dollars(amt.value);
    d.save.disabled = true;
    try { await api(path, b); d.close(); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); sync(); }
  });
  sync();
  d.open(name);
}

// modelDialog: one model's name and switches, from its row's Edit button
// (on a phone-width window the table keeps only the name and Edit).
function modelDialog(m) {
  const cur = modelName(m);
  const d = dialog(t("editModel", cur));
  const sec = section(d.body);
  const nl = el("label", t("name")); nl.htmlFor = "dlg-model-name";
  const name = el("input"); name.id = "dlg-model-name"; name.maxLength = 64; name.value = cur;
  sec.append(nl, name);
  const sw = (checked, label) => {
    const box = el("input"); box.type = "checkbox"; box.className = "switch"; box.checked = checked;
    const l = el("label", undefined, "toggle wrap"); l.append(box, el("span", label));
    return [l, box];
  };
  const [sl, shown] = sw(!m.hidden, t("showOnDevices"));
  const [hl, human] = sw(!!m.human_only || isSub(m), t("humanOnly"));
  human.disabled = isSub(m);
  const [al, inAuto] = sw(!m.no_auto, t("inAuto"));
  al.title = t("inAutoTip");
  const opts = section(d.body);
  opts.append(sl, al, hl);
  if (isSub(m)) opts.append(el("p", t("subAlways"), "muted small"));
  // Only what changed is sent; an unchanged name is not a rename.
  const diff = () => {
    const b = {};
    if (name.value.trim() !== cur) b.name = name.value.trim();
    if (shown.checked === !!m.hidden) b.shown = shown.checked;
    if (inAuto.checked === !!m.no_auto) b.auto = inAuto.checked;
    if (!isSub(m) && human.checked !== !!m.human_only) b.human_only = human.checked;
    return b;
  };
  d.dirty = () => Object.keys(diff()).length > 0;
  const sync = () => { d.save.disabled = !d.dirty(); };
  d.dlg.addEventListener("input", sync);
  d.dlg.addEventListener("change", sync);
  d.save.addEventListener("click", async () => {
    d.save.disabled = true;
    const b = diff();
    if (b.human_only === true && !(await confirmChatOnly(m))) { sync(); return; }
    try { await api("models/set", { id: m.id, ...b }); d.close(); toast(t("saved")); await refresh(); } catch (e) { toast(e.message, true); sync(); }
  });
  sync();
  d.open(name);
}

// chatOnlyLosers: the apps and programs-only devices that use model m
// now and could not once it is chat only, each with whether it is the
// only model it has (then it stops working entirely).
function chatOnlyLosers(m) {
  const uses = (list, id) => !(list || []).length || list.includes(id);
  const out = [];
  for (const a of S.apps || []) {
    if (!uses(a.Models, m.id)) continue;
    const left = S.models.filter((x) => x.id !== m.id && usableByApp(x) && uses(a.Models, x.id));
    out.push({ name: a.Name, kind: t("coApp"), only: left.length === 0 });
  }
  for (const d of S.devices || []) {
    if (!d.BackgroundOnly || !uses(d.Models, m.id) || m.hidden) continue;
    const left = S.models.filter((x) => x.id !== m.id && usableByDevice(d, x) && uses(d.Models, x.id));
    out.push({ name: d.Name, kind: t("coDevice"), only: left.length === 0 });
  }
  return out;
}

// confirmChatOnly asks before model m becomes chat only when that cuts
// someone off, naming them; true to go ahead. Nobody cut off: no question.
function confirmChatOnly(m) {
  const losers = chatOnlyLosers(m);
  if (!losers.length) return Promise.resolve(true);
  return new Promise((resolve) => {
    let done = false;
    const finish = (v) => { if (!done) { done = true; resolve(v); } };
    const d = dialog(t("coTitle", modelName(m)));
    d.body.append(el("p", t("coBody", modelName(m))));
    const ul = el("ul", undefined, "co-list");
    for (const x of losers) {
      const li = el("li");
      li.append(el("span", x.name, "co-name"), el("span", x.kind, "tag"));
      if (x.only) li.append(el("span", t("coOnly"), "co-only"));
      ul.append(li);
    }
    d.body.append(ul, el("p", t("coHow"), "muted small"));
    d.save.textContent = t("coYes");
    d.save.className = "danger";
    d.save.addEventListener("click", () => { finish(true); d.close(); });
    // Closed any other way (Cancel, ×, Escape, outside): nothing changes.
    const close = d.close;
    d.close = () => { close(); finish(false); };
    d.open();
    // Cancel was bound to the first close: tell it too.
    d.dlg.querySelector(".dialog-foot .secondary")?.addEventListener("click", () => finish(false));
  });
}

// modelName: a model as people see it here: the owner's own name as it
// is; else Codex is "ChatGPT (Codex · latest)", and a model that follows its
// family's newest version (not known yet) says so, in the page's language.
function modelName(m) {
  if (m.renamed) return m.display_name;
  // Codex names the version it runs only when the hub passes one on.
  if ((m.kind || kindOf(m.provider)) === "codex") return m.version ? t("codexVer", m.version) : t("codexName"); // same as the phone and the CLI
  return m.display_name + (m.latest ? t("latest") : "");
}
// withDot: "a · b", with no space after a full-width bracket
// ("ChatGPT（Codex）· 订阅").
const withDot = (a, b) => a + (/[）」』】]$/.test(a) ? "· " : " · ") + b;
// sortModels: the models of one source stay together, in the order the
// sources come; Claude's go Opus, Sonnet, Haiku; Codex's as Codex lists them.
function sortModels(ms) {
  const src = [...new Set(ms.map((m) => m.provider))];
  const rank = (m) => { const id = m.id.toLowerCase(); return id.includes("opus") ? 0 : id.includes("sonnet") ? 1 : id.includes("haiku") ? 2 : 3; };
  // Codex's models in Codex's own order (rank from its catalog).
  ms.sort((a, b) => src.indexOf(a.provider) - src.indexOf(b.provider) || rank(a) - rank(b) || (a.rank || 0) - (b.rank || 0));
}
// vendorChip: where a model's messages go: this Mac, or the company that
// runs it in the cloud.
function vendorChip(m) {
  if (m.locality === "local") return el("span", t("localChip"), "tag local");
  const k = kindOf(m.provider);
  const v = k === "claude-code" || k === "anthropic" ? "Anthropic" : k === "codex" || k === "openai" ? "OpenAI" : k === "antigravity" ? "Google" : t("remote");
  return el("span", v, "tag");
}

// aboutText: a wait in words, rounded the way people say it.
function aboutText(sec) {
  const n = sec >= 3600 ? Math.round(sec / 3600) : Math.max(1, Math.round(sec / 60));
  if (sec >= 3600) return n === 1 ? t("hour") : t("hours", n);
  return n === 1 ? t("minute") : t("minutes", n);
}
// problemText: what is wrong with a model source right now, if anything:
// signed out, or at its usage limit until resets_at (unix seconds).
function problemText(p) {
  if (p.problem === "login") return p.kind === "codex" ? t("problemLoginCodex") : p.kind === "antigravity" ? t("problemLoginAgy") : t("problemLogin");
  if (p.problem === "limit") {
    const sec = p.resets_at ? p.resets_at - Date.now() / 1000 : 0;
    return sec > 0 ? t("problemLimit", aboutText(sec)) : t("problemLimitNow");
  }
  return "";
}

function switchCell(tr, checked, label, onChange) {
  const td = cell(tr, "", "sw");
  const box = el("input"); box.type = "checkbox"; box.className = "switch"; box.checked = checked;
  box.setAttribute("aria-label", label);
  box.addEventListener("change", () => onChange(box));
  td.append(box);
  return box;
}

function renderModels() {
  const tb = $("models");
  tb.textContent = "";
  for (const m of S.models) {
    const tr = el("tr", undefined, m.source_off ? "paused" : "");
    const name = modelName(m);
    // The sources table below says which are subscriptions.
    const nc = cell(tr, name, "nm");
    // On a phone the chip sits under the name (its own column is hidden).
    nc.append(el("br", undefined, "narrow-only"), vendorChip(m)); nc.lastChild.classList.add("narrow-only");
    cell(tr, "").append(vendorChip(m));
    // A subscription is chat-only always: a word, not a switch that can't move.
    if (isSub(m)) { const c = cell(tr, "", "sw"); c.append(el("span", t("always"), "muted small")); c.title = t("subAlways"); }
    else switchCell(tr, !!m.human_only, t("humanOnly") + " " + name, async (box) => {
      // Turning it on cuts off apps and programs-only devices that use it:
      // say who first, and only go on when the owner says so.
      if (box.checked && !(await confirmChatOnly(m))) { box.checked = false; return; }
      try { await api("models/set", { id: m.id, human_only: box.checked }); toast(t("saved")); m.human_only = box.checked; } catch (e) { box.checked = !box.checked; toast(e.message, true); }
    });
    switchCell(tr, !m.hidden, t("showOnDevices") + " " + name, async (box) => {
      try { await api("models/set", { id: m.id, shown: box.checked }); toast(t("saved")); m.hidden = !box.checked; } catch (e) { box.checked = !box.checked; toast(e.message, true); }
    });
    switchCell(tr, !m.no_auto, t("inAuto") + " " + name, async (box) => {
      try { await api("models/set", { id: m.id, auto: box.checked }); toast(t("saved")); m.no_auto = !box.checked; } catch (e) { box.checked = !box.checked; toast(e.message, true); }
    });
    tr.lastChild.title = t("inAutoTip");
    // Edit, at every width: the name and both switches in a dialog (a
    // phone-width window keeps only the name and this).
    const ed = el("button", t("edit"), "secondary small-btn"); ed.type = "button";
    ed.addEventListener("click", () => modelDialog(m));
    cell(tr, "", "actions").append(ed);
    tb.append(tr);
  }
  const pb = $("providers");
  pb.textContent = "";
  for (const p of S.providers) {
    const tr = el("tr", undefined, p.off ? "paused" : "");
    // Sources by the names people know them by; "(this Mac)" is in
    // Ollama's own name.
    const own = sourceName(p.kind, p.name), label = own || p.display_name || p.name;
    const how = p.subscription ? t("subLogin") : p.key_last4 ? t("key") + " …" + p.key_last4 : t("local");
    // On a phone-width window the row stacks: the name, then how it is
    // connected; Remove on the right.
    const nc = cell(tr, "", "src");
    nc.append(el("div", p.local && !own ? withDot(label, t("local")) : label, "src-name"));
    nc.append(el("div", how, "nm-sub narrow-only"));
    const pt = problemText(p);
    if (pt) nc.append(el("span", pt, "tag warn-tag"));
    // Signed out: the sign-in right here (the phone and the error point to it).
    if (p.problem === "login" && p.subscription) {
      const b = el("button", t("subSignIn"), "link"); b.type = "button";
      b.addEventListener("click", async () => {
        // Not a Mac: no window to open here; the card shows the command.
        if (!onMac()) { fromDash = true; return showWizard(); }
        b.disabled = true; // one Terminal window per click
        try { await api("subs/login", { kind: p.kind }); toast(t("subHow")); } catch (e) { toast(e.message, true); }
        setTimeout(() => { b.disabled = false; }, 5000);
      });
      nc.append(" ", b);
    }
    cell(tr, how, "how");
    // Off keeps the login or key; Remove (quieter) deletes it.
    pauseSwitch(tr, label, "providers/off", p.name, !!p.off, "off", ["sourceOffToast", "sourceOnToast"]);
    const rm = removeButton(label, "providers/remove", p.name, renderModels, "removeConfirmSource");
    rm.className = "danger-text";
    cell(tr, "", "actions").append(rm);
    pb.append(tr);
  }
}

function renderApps() {
  // Where programs connect, or that they can't yet.
  const where = $("apps-where");
  where.textContent = "";
  where.hidden = !!S.api_url;
  if (!S.api_url) where.append(el("span", t("appsOff"), "tag warn-tag"));
  $("apps-help").hidden = !S.api_url;
  const tb = $("apps");
  tb.textContent = "";
  for (const a of S.apps) {
    const tr = el("tr", undefined, a.Paused ? "paused" : "");
    const spent = spentText("app/" + a.ID, a.BudgetMicroUSD);
    const nc = nameCell(tr, a.Name, a.Paused, false, spent);
    hoverCard(nc.querySelector(".nm-text"), () => nameCard(a, "app", spent));
    // Nothing it may use works for apps (only subscriptions or chat-only
    // models are connected, or its own list has none left).
    // On a phone-width window it always takes a line of its own.
    // The chip leads to where that is fixed: Models › Connect a model.
    if (!appHasModels(a)) {
      const chip = el("a", t("appNoModels"), "tag warn-tag chip-link");
      chip.href = "#"; chip.title = t("appNoModelsTip");
      chip.addEventListener("click", (e) => { e.preventDefault(); document.querySelector('.tabs button[data-tab="models"]').click(); $("add-provider").click(); });
      nc.firstChild.append(el("span", "", "brk narrow-only"), chip);
    }
    cell(tr, spent, "spent");
    modelsCell(tr, nc, a, usableByApp);
    pauseSwitch(tr, a.Name, "apps/update", a.ID, a.Paused);
    const ed = el("button", t("edit"), "secondary small-btn");
    ed.type = "button";
    ed.addEventListener("click", () => editDialog("app", a));
    cell(tr, "", "actions").append(ed);
    tb.append(tr);
  }
  $("apps-empty").hidden = S.apps.length > 0;
}

// The Usage tab's range, kept for next time: today, yesterday or week.
let usageRange = ["today", "yesterday", "week"].includes(store("omnillm.range")) ? store("omnillm.range") : "today";
// rangeWords: the total line and empty line of the range the data is for.
const rangeWords = (r) => r === "yesterday" ? [t("totalYesterday"), t("emptyYesterday")]
  : r === "week" ? [t("totalWeek"), t("emptyWeek")] : [t("totalToday"), t("emptyToday")];
const pressRange = () => document.querySelectorAll("#usage-range button").forEach((b) => b.setAttribute("aria-pressed", String(b.dataset.range === usageRange)));
// A click shows the new choice at once; the tables change when that
// range's numbers arrive (an older answer that comes later is dropped).
document.querySelectorAll("#usage-range button").forEach((b) => {
  b.addEventListener("click", async () => {
    if (b.dataset.range === usageRange) return;
    usageRange = b.dataset.range;
    store("omnillm.range", usageRange);
    pressRange();
    try { await refresh(); } catch (e) { toast(e.message, true); }
  });
});

// renderUsage: one line per device or app by its name (never an internal
// id), busiest first; removed devices folded into one line; the range's
// total at the bottom. Below it the same per model.
const num = (n) => Number(n || 0).toLocaleString(lang === "zh" ? "zh-CN" : "en-US");
function renderUsage() {
  pressRange();
  const [totalLabel, emptyText] = rangeWords(S.range);
  $("usage-body").textContent = t("usageBody");
  const tb = $("usage"), foot = $("usage-total");
  tb.textContent = ""; foot.textContent = "";
  const paused = new Set([...S.devices.filter((d) => d.Paused).map((d) => "device/" + d.ID), ...S.apps.filter((a) => a.Paused).map((a) => "app/" + a.ID)]);
  const rows = [], gone = { calls: 0, micro_usd: 0, input_tokens: 0, output_tokens: 0, n: 0, name: "" };
  let total = null;
  for (const u of S.range_usage || []) {
    if (u.who === "_total") { total = u; continue; }
    if (u.removed) {
      gone.n++; gone.name = u.name || "";
      for (const k of ["calls", "micro_usd", "input_tokens", "output_tokens"]) gone[k] += u[k] || 0;
      continue;
    }
    rows.push({ ...u, label: u.who === "host" ? t("usageHost") : (u.name || u.who), paused: paused.has(u.who) });
  }
  rows.sort((a, b) => b.calls - a.calls);
  if (gone.n) {
    const one = gone.n === 1 && gone.name;
    rows.push({ ...gone, label: one ? t("usageGoneOne", gone.name) : t("usageGone"), count: one ? 0 : gone.n, short: one ? "" : t("usageGoneShort"), muted: true });
  }
  const line = (u, into, cls) => {
    const tr = el("tr", undefined, cls);
    const c = cell(tr, "", u.muted ? "muted nm" : "nm");
    // "Removed devices and apps (23)": the count never breaks off. On a
    // phone-width window the removed ones' line is short: "Removed (23)".
    if (u.short) c.append(el("span", u.label, "wide-only"), el("span", u.short, "narrow-only"));
    else c.append(u.label);
    if (u.count) c.append(lang === "zh" ? "" : " ", el("span", t("goneCount", num(u.count)), "nowrap"));
    if (u.paused) c.append(" ", el("span", t("pausedChip"), "tag paused-tag"));
    cell(tr, num(u.calls), "num");
    cell(tr, money(u.micro_usd), "num");
    cell(tr, num(u.input_tokens) + " / " + num(u.output_tokens), "num narrow-hide");
    into.append(tr);
  };
  rows.forEach((u) => line(u, tb));
  if (total && rows.length) line({ ...total, label: totalLabel }, foot, "total");
  $("usage-empty").textContent = emptyText;
  $("usage-empty").hidden = rows.length > 0;
  // By model: the same range, one line per model by its name.
  const mt = $("usage-models"), mfoot = $("usage-models-total");
  mt.textContent = ""; mfoot.textContent = "";
  const mrows = [], mgone = { calls: 0, micro_usd: 0, input_tokens: 0, output_tokens: 0, n: 0 };
  for (const u of S.model_usage || []) {
    if (u.removed) {
      mgone.n++;
      for (const k of ["calls", "micro_usd", "input_tokens", "output_tokens"]) mgone[k] += u[k] || 0;
      continue;
    }
    const m = S.models.find((x) => x.id === u.who);
    mrows.push({ ...u, label: m ? modelName(m) : (u.name || u.who) });
  }
  mrows.sort((a, b) => b.calls - a.calls);
  if (mgone.n) mrows.push({ ...mgone, label: t("modelsGone"), count: mgone.n, muted: true });
  // Calls from before use was kept per model are in the total above but in
  // no model's line: one line says so, so the two totals agree.
  const fields = ["calls", "micro_usd", "input_tokens", "output_tokens"];
  const sum = Object.fromEntries(fields.map((k) => [k, mrows.reduce((n, u) => n + (u[k] || 0), 0)]));
  if (total && mrows.length && (total.calls > sum.calls || total.micro_usd > sum.micro_usd)) {
    mrows.push({ ...Object.fromEntries(fields.map((k) => [k, Math.max(0, (total[k] || 0) - sum[k])])), label: t("byModelEarlier"), muted: true });
  }
  mrows.forEach((u) => line(u, mt));
  if (mrows.length) line({ ...(total || sum), label: totalLabel }, mfoot, "total");
  $("usage-models-box").hidden = mrows.length === 0;
}

document.querySelectorAll(".tabs button").forEach((b) => {
  b.addEventListener("click", () => {
    document.querySelectorAll(".tabs button").forEach((x) => x.setAttribute("aria-selected", String(x === b)));
    document.querySelectorAll(".tab").forEach((x) => { x.hidden = x.id !== "tab-" + b.dataset.tab; });
    store("omnillm.tab", b.dataset.tab);
  });
});

$("add-device").addEventListener("click", () => startPair($("pair-dash"), (id) => { closePair($("pair-dash")); afterPair(id); }, true));

// afterPair: the new device opens its session a moment after it is
// added; look again (every second, for up to 10 s) until it shows online.
// The first redraw that has it scrolls its row into view and lights it up
// for 2 s.
function afterPair(id) {
  let n = 0, shown = false;
  const look = async () => {
    // The owner started something meanwhile (editing, adding another
    // device): don't redraw under them; just look again in a second.
    if (busy()) { if (++n < 10 && !signedOut) setTimeout(look, 1000); return; }
    if (!shown) fresh = { id, until: Date.now() + 2000 };
    try { await refresh(); } catch (_) { return; }
    const row = !shown && document.querySelector('#devices tr[data-id="' + CSS.escape(id) + '"]');
    if (row) {
      shown = true;
      row.scrollIntoView({ block: "center", behavior: matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" });
      setTimeout(() => { fresh = { id: "", until: 0 }; document.querySelectorAll("#devices tr.fresh").forEach((r) => r.classList.remove("fresh")); }, 2000);
    }
    const d = S.devices.find((x) => x.ID === id);
    if (d && !d.Online && ++n < 10 && !signedOut) setTimeout(look, 1000);
  };
  look();
}
// The wizard's first step shows while S.providers is empty; how many are
// really connected is kept for "Check again".
$("add-provider").addEventListener("click", () => { fromDash = true; showWizard(); });
$("wiz-back").addEventListener("click", async () => { fromDash = false; store("omnillm.wizard", "done"); try { await refresh(); } catch (e) { toast(e.message, true); } });
// "Add app": a name field in place, then the app's token, shown this once
// with its Copy button.
const appForm = $("app-add"), appName = $("app-name");
const closeAppForm = () => { appForm.hidden = true; $("add-app").hidden = false; appName.value = ""; };
clickCard($("apps-help"), appsHelpCard);
$("add-app").addEventListener("click", () => {
  appForm.hidden = false; $("add-app").hidden = true; $("app-add-ok").disabled = true;
  appName.focus();
});
appName.addEventListener("input", () => { $("app-add-ok").disabled = !appName.value.trim(); });
appName.addEventListener("keydown", (e) => { if (composing(e)) return; if (e.key === "Escape") closeAppForm(); });
noImeSubmit(appForm);
$("app-add-cancel").addEventListener("click", closeAppForm);
appForm.addEventListener("submit", async (ev) => {
  ev.preventDefault();
  const name = appName.value.trim();
  if (!name) return;
  const ok = $("app-add-ok");
  ok.disabled = true;
  try {
    const r = await api("apps/add", { name });
    closeAppForm();
    const box = $("app-token");
    box.textContent = "";
    box.append(el("p", t("appTokenFor", r.name), "token-for"), codeBox(r.token), el("p", t("appTokenOnce"), "muted small"));
    // How to use it, ready to copy: the address and the token together.
    if (S.api_url) {
      box.append(el("p", t("appTokenUse"), "small token-use"),
        codeBox("OPENAI_BASE_URL=" + S.api_url + "\nOPENAI_API_KEY=" + r.token),
        el("p", t("appTokenTry"), "small token-use"),
        codeBox("curl " + S.api_url + "/models -H \"Authorization: Bearer " + r.token + "\"", true));
    }
    box.hidden = false;
    refresh();
  } catch (e) { toast(e.message, true); ok.disabled = false; }
});

// The language is the owner's choice for the console and the menu bar
// together: the hub keeps it, so a switch here changes the menu bar, and
// one made in the menu bar shows here within a few seconds (followLang).
function useLang(l) {
  lang = l; store("omnillm.lang", lang); applyLang();
  if (S) { renderStatus(); if (!$("dash").hidden) showDash(); }
  if (S && !$("wizard").hidden) { if (!$("w1").hidden) renderSubs(); if (!$("w3").hidden) fillModels(); }
  if (pairRedraw) pairRedraw();
}
// followLang takes the hub's choice when it differs from the page's; true
// when it redrew.
function followLang(l) {
  if ((l !== "zh" && l !== "en") || l === lang) return false;
  if (S) S.lang = l;
  useLang(l);
  return true;
}
// Back in view (polls pause while hidden): pick up a language chosen in
// the menu bar meanwhile at once.
document.addEventListener("visibilitychange", async () => {
  if (document.hidden || !S || signedOut) return;
  try { followLang((await api("relay")).lang); } catch (_) {}
});
$("lang").addEventListener("click", () => {
  const l = lang === "zh" ? "en" : "zh";
  if (S) S.lang = l;
  useLang(l);
  api("lang", { lang: l }).catch(() => {});
});
$("theme").addEventListener("click", () => {
  const cur = document.documentElement.dataset.theme || (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
  const next = cur === "dark" ? "light" : "dark";
  store("omnillm.theme", next);
  applyTheme(next);
});

document.querySelectorAll("button[data-copy]").forEach((b) => copyBtn(b.dataset.copy, b));

let bootTries = 0;
async function boot() {
  applyLang();
  applyTheme(store("omnillm.theme"));
  // The menu bar's counts open their tab with &tab=devices / &tab=models.
  const ht = /[#&]tab=(\w+)/.exec(location.hash);
  const tab = (ht && ht[1]) || store("omnillm.tab") || "devices";
  const tb = document.querySelector('.tabs button[data-tab="' + tab + '"]') || document.querySelector(".tabs button");
  tb.click();
  const m = /(?:^|[#&])login=([0-9a-f]{64})/.exec(location.hash);
  // "Add a phone or computer…" in the menu bar opens the console with &add.
  const wantAdd = /[#&]add(?:&|$)/.test(location.hash);
  // The installer opened the sign-in window for this subscription: its
  // card waits for that sign-in (and looks every few seconds).
  const si = /[#&]signin=(claude-code|codex|antigravity)(?:&|$)/.exec(location.hash);
  if (si) subsWaiting[si[1]] = Date.now();
  if (m) {
    history.replaceState(null, "", "/console/");
    try { const r = await api("login", { token: m[1] }); sessionSecret(r.session); } catch (e) { show("login-needed"); return; }
  }
  try { await refresh(); } catch (e) {
    if (e.status === 401) show("login-needed");
    // Restarting (an update): try again, less often, then stop; only
    // when it didn't answer or failed on its side.
    else if ((e.status === undefined || e.status >= 500) && bootTries < 8) { bootTries++; setTimeout(boot, Math.min(30000, 3000 * bootTries)); }
    else toast(e.message, true);
    return;
  }
  if (wantAdd && !$("dash").hidden) {
    document.querySelector('.tabs button[data-tab="devices"]').click();
    $("add-device").click();
  }
}
boot();
