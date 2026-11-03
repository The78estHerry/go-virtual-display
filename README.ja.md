<p align="center"><img src="assets/icon-256.png" width="160" alt="go-virtual-display のアイコン：夢の吹き出しの中で虹色のモニターを思い浮かべている"></p>

# go-virtual-display

[繁體中文](README.md) | [English](README.en.md) | **日本語**

> 正式な説明は繁體中文版の README です。この翻訳と内容が異なる場合は、中国語版が正しい内容です。

**物理モニターを接続していない**状態でも、モニターが接続されているとパソコンに認識させます。このモニターは、本プロジェクトで生成した EDID を返します。解像度は選択・追加・削除でき、既定は 1920×1080 です。あわせて「抜き差しスイッチ」も付属しています。タスクトレイのアイコンとデスクトップウィジェットから、ワンクリックで仮想モニターを接続・切断できます。

ヘッドレス環境でのリモートデスクトップ、配信、録画や、特定の解像度が必要なテストに向いています。

**Windows 10／11** と **Linux**（グラフィックカードに空いている端子が必要）に対応しています。macOS には対応していません。

Windows 版は **Synergy 3**（マウス・キーボード共有ソフト）に対応しています。画面構成が変わると Synergy を自動で再起動し、Synergy が動作中かどうかをタスクトレイとウィジェットに表示します（「[Synergy 3 との併用](#synergy-3-との併用)」を参照）。

> 本プログラムの画面表示は英語と繁体字中国語のみです。日本語の画面表示はありません。メニュー名などは英語版の表記で記載しています。

## 仕組み

| | Windows | Linux |
|---|---|---|
| 仮想モニターの実体 | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver)（署名済みのオープンソース IddCx ドライバー、MIT） | カーネル内蔵の DRM EDID 上書き機能で、空き端子にモニターが接続されているように見せる |
| カスタム EDID | ドライバーが `C:\VirtualDisplayDriver\user_edid.bin` を読み込む | 起動パラメーター `drm.edid_firmware=` で `/lib/firmware/edid/go-virtual-display.bin` を指定。解像度の切り替え時は debugfs の `edid_override` に書き込んで即時反映 |
| 抜き差し | ドライバーのデバイスを無効化／有効化（`pnputil`）。「最上位の特権」のタスクで実行するので、スイッチ操作で UAC は出ない | `/sys/class/drm/<端子>/status` に `on`／`off` を書き込む。polkit により、ログイン中のユーザーはパスワードなしで実行できる |

ドライバー本体は**このリポジトリには含まれていません**。
- **Windows のリリースパッケージ**には固定バージョン（25.7.23）のドライバーファイルが同梱されており、インストール時に SHA-256 を検証します。**インターネット接続は不要**で、元のダウンロード先がなくなってもインストールできます。
  - 同梱しているのは必要なファイルだけです（amd64／ARM64 のドライバーと `devcon.exe`）。内容は公式の zip と同一なので、署名も有効なままです。
  - ライセンス表記はパッケージ内の `scripts\vdd\THIRD-PARTY-NOTICES.txt` にあります（「[サードパーティソフトウェア](#サードパーティソフトウェア)」を参照）。
- **ソースコードからインストールする場合**は、公式の zip をダウンロードして SHA-256 を検証します。
- 各ドライバーファイルの SHA-256 は `windows\scripts\common.ps1` に記載されているので、パッケージ内のファイルが改ざんされても検出できます。

## EDID の内容

解像度ごとに EDID を 1 つ生成します（`internal/profile`）。

- EDID 1.4、128 バイト、拡張ブロックなし
- 推奨タイミングは VESA CVT reduced blanking（最近の液晶モニターで一般的なタイミング）。1920×1440@60 だけは例外で、旧バージョンと同じ VESA DMT を使用
- 名前は `VirtualDisp`、シリアル文字列は `GVD-<幅>x<高さ>`（例：`GVD-1920x1080`）、sRGB
- 物理サイズは 96 DPI で換算（1920×1080 → 51×29 cm）。Windows の既定の拡大率は 100% になる
- Virtual Display Driver はメーカー／製品コードを `MTT`／`0x1337` に置き換えるため、解像度の区別にはシリアル文字列を使う。Windows は解像度ごとに配置と拡大率を記憶する

リポジトリには生成済みの EDID が 2 つあります。
- `edid/edid_1920x1080.bin`：インストール時の既定（Windows、Linux）
- `edid/edid_1920x1440.bin`：0.3.0 より前の Linux 版で使用。DMT タイミングの参考として残している

```
go run ./cmd/make-edid -w 2560 -h 1440 -hz 60 -o out.bin   # 指定した解像度で生成
go run ./cmd/make-edid -check edid/edid_1920x1080.bin      # 検査して内容を表示
```

## 解像度

- **切り替え**：タスクトレイのアイコンを右クリック →「Resolution」で、使いたい解像度をクリックします。仮想モニターが一瞬再起動します。Windows で Synergy がインストールされている場合は、Synergy も自動で再起動します。切断中に切り替えた場合は、次に接続したときに反映されます。
- **管理**：同じサブメニューの「Manage resolutions…」でウィンドウが開きます。
  - 保存済みの解像度は「Use」（使用）または「Delete」（削除）できます。使用中のものは削除できません
  - よく使う解像度はワンクリックで追加できます（すべて 60 Hz）：1920×1080、2560×1440、3840×2160、1280×720、1920×1200、2560×1600、3840×2400、640×480、1024×768、1600×1200、1920×1440、1280×1024、2560×1080、1080×1920
  - 幅・高さ・リフレッシュレートを手入力することもできます。幅と高さは 320〜4095、リフレッシュレートは 24〜240 Hz。高すぎて EDID で表せない場合はエラーになります
- インストール直後は 1920×1080@60 のみです。一覧は `%APPDATA%\go-virtual-display\profiles.json`（Linux は `~/.config/go-virtual-display/profiles.json`）に保存されます。上書きインストール（アップグレードや再インストール）では保持され、アンインストール時に削除されます。
- **Windows**：切り替えは最上位の特権のタスク `\go-virtual-display\Apply` が `vdswitch.exe --apply` を実行して行います。EDID の生成と `C:\VirtualDisplayDriver` の設定の書き換えを行うので、UAC は表示されません。
- **Linux**：スイッチプログラムが EDID を生成し、polkit 経由でヘルパー（`vdswitch-helper edid`、パスワード不要）に渡します。ヘルパーは長さ・ヘッダー・チェックサムを検査したうえで `/lib/firmware/edid/go-virtual-display.bin` を置き換え、バックグラウンドで initramfs を再生成します。起動パラメーターは常にこのファイル名を指しているので、GRUB の変更は不要です。
  - debugfs（`/sys/kernel/debug/dri/<カード>/<端子>/edid_override`）がある場合は即時に反映されます
  - ない場合（Secure Boot の lockdown が有効なときなど）は、ウィジェットに「Reboot to apply」（再起動後に反映）と表示されます

## Windows

### 必要なもの

- Windows 10／11 x64（ARM64 用のドライバーもありますが、Windows 11 24H2 以降ではテスト署名モードが必要になる場合があります）
- [Microsoft Visual C++ 再頒布可能パッケージ](https://learn.microsoft.com/cpp/windows/latest-supported-vc-redist)（ない場合、ドライバーが `vcruntime140.dll not found` を報告します）
- Go がない場合は、Gitea の Releases から `go-virtual-display-<バージョン>-windows.zip` をダウンロードしてください。ビルド済みのスイッチプログラムが入っています。自分でビルドする場合は Go 1.26 以降が必要です

### ワンクリックインストール

**`windows\install.cmd`** をダブルクリックし、UAC の確認で「はい」を押します。あとはすべて自動で行われます。

1. 表示言語（English／繁體中文）を選択。Enter キーで既定値になります（下の「表示言語」を参照）
2. ドライバーを取得して SHA-256 を検証：パッケージでは同梱のものを使い、ソースコードの場合だけダウンロードします。この手順に失敗した場合は何も変更せずに中止します（再インストール時に旧バージョンを先に削除することもありません）
3. `vdd_settings.xml` と EDID を `C:\VirtualDisplayDriver` に配置（既存のファイルは先にバックアップ）。EDID は保存されている解像度から生成し、新規インストールでは 1920×1080
4. ドライバーをインストール
5. タスク（`\go-virtual-display\Plug`、`Unplug`、`Apply`、`RestartServices`）を作成。以降、スイッチ操作や解像度の切り替えで UAC は表示されません。Synergy がインストールされている場合は、画面構成が変わると自動で再起動します（後述）
6. スイッチプログラムを `C:\Program Files\go-virtual-display` にインストールし、スタートメニューとデスクトップに「Virtual Display Switch」のショートカットを作成（タスクバーにピン留めしたい場合は、ショートカットを右クリックして「タスクバーにピン留めする」を選んでください）。まだビルドされておらず Go がある場合は、先に自動でビルドします
7. アンインストール用のファイルをコピーし、「設定 > アプリ」に登録
8. チェックを実行し、スイッチプログラムを起動

**すでにインストールされている場合**（アップグレードや再インストール）は、現在のバージョンとインストールするバージョンを表示し（ダウングレードの場合は注意を表示）、続行するかどうかを確認します。
- `y` と答える：現在のバージョンをアンインストールしてから新規インストールします。追加した解像度、選んだ言語、「Start at sign-in」（サインイン時に自動起動）は保持されます
- そのまま Enter またはそれ以外の答え：キャンセルし、何も変更しません

### アンインストール

次のいずれかの方法で行います。どれも UAC の確認が出るので「はい」を押してください。
- タスクトレイのアイコンを右クリックし、「**Uninstall…**」を選択
- 「**設定 > アプリ**」で「Virtual Display Switch (go-virtual-display)」（中国語表示の場合は「虛擬螢幕開關（go-virtual-display）」）を探してアンインストール
- `windows\uninstall.cmd` をダブルクリック

インストール時にアンインストール用のファイルを `C:\Program Files\go-virtual-display\uninstaller` にコピーしているので、最初に展開したフォルダーを削除しても、上の 2 つの方法でアンインストールできます。アンインストールでは次の処理を行います。
- ドライバー、タスク、スイッチプログラム、サインイン時の自動起動の設定を削除
- Windows が残した仮想モニターのデバイス記録、拡大率の設定、VDD のレジストリ値を削除
- 元の VDD 設定ファイルを復元。インストール前になかった場合は、`C:\VirtualDisplayDriver` フォルダーごと削除
- ユーザー設定を削除：追加した解像度、ウィジェットの位置、選んだ言語（`%APPDATA%\go-virtual-display`、`HKLM\SOFTWARE\go-virtual-display`）。アップグレードだけなら、新しい `install.cmd` を実行するだけで十分で、先にアンインストールする必要はありません

### 手動での実行（上級者向け）

ワンクリックインストールの裏では `windows\scripts\` のスクリプトを実行しています。管理者の PowerShell で個別に実行することもできます。

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1                # ビルド（Go が必要）
powershell -ExecutionPolicy Bypass -File .\windows\scripts\install.ps1      # インストール
powershell -ExecutionPolicy Bypass -File .\windows\scripts\verify.ps1       # チェック（管理者不要）
powershell -ExecutionPolicy Bypass -File .\windows\scripts\uninstall.ps1    # アンインストール
```

### Synergy 3 との併用

Synergy 3（サービス名 `Synergy Core Daemon`）のクライアントで動作を確認しています。

- **自動再起動**：Synergy のクライアントは、起動時にしかデスクトップ全体のサイズを読み込みません。仮想モニターを接続・切断すると古いサイズのままマウス位置を計算するため、サーバーから移ってきたマウスが別のモニターに出てしまいます。「設定 > ディスプレイ」でモニターの配置を変えた後も同様で、マウスが古い端で止まってしまいます。
  - スイッチプログラムが画面構成の変化（抜き差し、解像度の切り替え、配置の変更、ほかのモニターの接続）を検出すると、約 2 秒待って安定してから、タスク `RestartServices` で Synergy を再起動します。Synergy は一瞬切断され、自動で再接続します。
  - このタスクは常に作成され、実行時に Synergy の有無を確認します。そのため、あとから Synergy をインストール・アンインストールしても、本プログラムを再インストールする必要はありません。
  - スイッチプログラムが起動していないときは処理されません。
- **状態表示**：Synergy がインストールされている場合、ウィジェットの状態の下に「Synergy 3 Running」（実行中）／「Synergy 3 Not running」（未実行）の行が加わり、タスクトレイのメニューにも「Synergy 3: Running」／「Synergy 3: Not running」が表示されます（数字は Synergy のメジャーバージョンで、`Synergy.exe` のバージョン情報から読み取ります。読み取れない場合は「Synergy」とだけ表示）。どちらも Synergy のアイコン付きで、実行中はカラー、未実行はグレーです。インストールされていなければ表示しません。
  - 「実行中」とは、Synergy のサービスと、実際にマウス・キーボードを共有する `synergy-core.exe` の両方が動いている状態です。Synergy 自身のタスクトレイから Synergy を終了すると、サービスは残りますが `synergy-core.exe` は止まるので、「Not running」と表示されます。
  - Synergy が未実行のときは、画面構成が変わっても再起動しません。終了した Synergy が勝手に起動しないようにするためです。
  - アイコンは、お使いのパソコンにインストールされている Synergy から実行時に読み込んでいます。本プロジェクトは Synergy の画像ファイルを同梱していません。
- 同じく起動時にしかデスクトップのサイズを読み込まないほかのサービスは、`windows\scripts\common.ps1` の `$RestartServices` に追加できます。

本プロジェクトは Synergy の開発元である Symless とは一切関係がなく、承認や後援も受けていません（「[商標](#商標)」を参照）。

> ⚠️ グラフィックドライバーやチップセットドライバーを更新する前に、仮想ディスプレイドライバーをアンインストールしておくことをおすすめします（VDD 公式の推奨）。起動後に画面が真っ暗になった場合は、セーフモードでアンインストールしてください。

## Linux

### 必要なもの

- グラフィックカードに**モニターが接続されていない**端子（例：HDMI-A-1）。実際のグラフィックカードがない仮想マシンやサーバーでは使えません。
- polkit（`pkexec`）
- タスクトレイのアイコンには、デスクトップの StatusNotifierItem 対応が必要です。KDE はそのまま対応、GNOME は AppIndicator 拡張機能が必要です
- 自分でビルドする場合は、Go と Gio の依存パッケージが必要です（`Makefile` の冒頭を参照）
- 起動パラメーターの自動設定は GRUB に対応。initramfs の自動設定は initramfs-tools（Debian／Ubuntu）と dracut（Fedora など）に対応しています

### ワンクリックインストール

一般ユーザーでターミナルから実行します（sudo は不要で、必要なときにパスワードを尋ねます）。

```bash
linux/install.sh        # ソースコードから
./install.sh            # リリースパッケージから（go-virtual-display-<バージョン>-linux-amd64.tar.gz を展開した後）
```

あとはすべて自動で行われます。

1. まだビルドされておらず Go がある場合は、先にスイッチプログラムをビルド（ソースコードのみ。パッケージにはビルド済みのものが入っています）
2. モニターが接続されていない外部端子（HDMI、DP など。ノートパソコンの内蔵パネルは選ばない）を自動で選択。指定したい場合は `linux/install.sh card0-HDMI-A-1` のようにします
3. 保存されている解像度から EDID を生成（新規インストールでは 1920×1080）し、EDID、ヘルパー、polkit ルールをインストール。以降、抜き差しや解像度の切り替えでパスワードは不要です
4. EDID を initramfs に追加
5. `/etc/default/grub` に起動パラメーターを追加し、GRUB の設定を再生成。初回は `grub.go-virtual-display.bak` としてバックアップを作成
6. debugfs がある場合はその場で一度適用し、再起動は不要。ない場合は再起動が必要
7. スイッチプログラムを起動

**すでにインストールされている場合**（アップグレードや再インストール）は、現在のバージョンとインストールするバージョンを表示し（ダウングレードの場合は注意を表示）、続行するかどうかを確認します。
- `y` と答える：現在のバージョンをアンインストールしてから新規インストールします。追加した解像度、選んだ言語、端子、「Start at sign-in」はすべて保持されます
- そのまま Enter またはそれ以外の答え：キャンセルし、何も変更しません
- 確認できるターミナルがない場合（ほかのプログラムから実行した場合など）は、そのまま上書きインストールします

何度実行してもパラメーターが重複することはなく、選んだ解像度も保持されます。端子を変えた場合は、古い端子のパラメーターを削除します。0.3.0 より前のバージョンからアップグレードする場合は、古い `edid_1920x1440.bin` とその起動パラメーターを新しいファイル名に置き換えます。

### ワンクリックアンインストール

```bash
linux/uninstall.sh
```

インストールしたすべてのファイル、起動パラメーター、initramfs の設定、サインイン時の自動起動の設定、保存した解像度（`~/.config/go-virtual-display`）を削除し、端子を自動検出に戻します。削除するのは本プロジェクトが追加したパラメーターだけで、もともとの起動パラメーターには手を加えません。再起動すると完全に元の状態に戻ります。

チェックには `linux/scripts/verify.sh` を使います。

## アイコン

プログラムのアイコンは「夢の吹き出しの中でモニターを思い浮かべている」デザインで、画面には虹と雲が描かれています。`vdswitch.exe` に埋め込まれており、「インストールされているアプリ」、スタートメニュー、エクスプローラーに表示されます。タスクトレイのアイコンは、状態によって色が変わるモニターのままです（緑：接続、グレー：切断、オレンジ：切り替え中、赤：エラー）。

アイコンを変更するときは `assets/icon.svg` を編集し、`assets/make-icon.ps1` を実行します。Windows 標準の Edge で SVG を各サイズに変換して `icon.ico` を作り、llvm-mingw の windres で `cmd/vdswitch/rsrc_windows_amd64.syso` を生成します。生成物はコミットしてあるので、通常のビルドにこれらのツールは不要です。

## 表示言語

スイッチプログラム、インストールとアンインストールのメッセージは、英語と繁体字中国語の 2 種類です。日本語の表示はありません。

- **既定**：システムの表示言語で決まります。中国語（繁体字・簡体字を問わず）の場合だけ繁体字中国語になり、それ以外（日本語を含む）はすべて英語です。
- **インストール時に選択可能**：Windows の `install.cmd`、Linux の `install.sh` の最初に確認されます。Enter キーで既定値になります。再インストール時は、前回選んだ言語が既定値になります。
- **保存場所**：Windows はレジストリ `HKLM\SOFTWARE\go-virtual-display` の `Language`、Linux は `/etc/go-virtual-display.conf` の `LANGUAGE`（`en` または `zh-TW`）。アンインストール時に削除されます。
- ファイル名はすべて英語です（スタートメニューのショートカット「Virtual Display Switch」やパッケージ内の `README.txt` を含む）。中国語以外のシステムで問題が起きないようにするためです。

## 抜き差しスイッチ（vdswitch）

- **デスクトップウィジェット**：
  - 右側にスライドスイッチがある小さなカードで、クリックすると切り替わります
  - 緑は接続、グレーは切断、オレンジは切り替え中、赤はエラー（カードに原因を表示）
  - 左側の文字の部分をドラッグするとカードを移動できます
  - Windows はカードの位置を記憶します。Linux ではウィンドウマネージャーが位置を決めます
  - Windows：Synergy がインストールされている場合、状態の下に Synergy のアイコンと「Running」／「Not running」の行が加わり、カードが少し高くなります（「[Synergy 3 との併用](#synergy-3-との併用)」を参照）
  - Windows：カードがあるモニターが切断されたとき（メインディスプレイに設定した仮想モニター上にある場合など）は、カードを一時的にメインディスプレイの右下に移動し、そのモニターが再接続されると元の位置に戻します。その間に自分でドラッグした場合は、新しい位置のままになります
- **タスクトレイのアイコン**：
  - 色はウィジェットと同じです
  - **左クリックで切り替え**
  - 右クリックメニュー：Synergy の状態（Windows のみ、Synergy がインストールされている場合）、Plug in／Unplug（接続／切断）、Resolution（解像度、Windows のみ）、Show desktop widget（ウィジェットを表示）、Keep widget on top（ウィジェットを最前面に固定）、Start at sign-in（サインイン時に自動起動）、Uninstall…（Windows のみ）、Exit（終了）
- 状態は 2 秒ごとに同期します。ほかの場所（デバイスマネージャー、VDD のコントロールプログラム）で状態を変えた場合も、両方に反映されます。
- 起動できるのは 1 つだけです。もう一度起動すると、ウィジェットを表示します。
- 設定は `%APPDATA%\go-virtual-display\widget.json`（Linux は `~/.config/go-virtual-display/widget.json`）に保存されます。

## プロジェクト構成

```
cmd/make-edid/        EDID の生成／検査
cmd/vdswitch/         抜き差しスイッチ本体（多重起動防止を含む）
internal/edid/        EDID 1.4 の組み立てと解析（テストを含む）
internal/i18n/        表示言語の判定と英語／繁体字中国語の文言
internal/profile/     解像度の一覧（profiles.json）と各解像度の EDID
internal/display/     プラットフォームごとの状態取得と抜き差し
internal/ui/          タスクトレイ（fyne.io/systray）、ウィジェット（Gio）、設定、自動起動
edid/                 生成済みの EDID
assets/               プログラムのアイコン：icon.svg（元ファイル）、icon.ico、make-icon.ps1（.ico と cmd/vdswitch 用の .syso を生成）
windows/              Windows のワンクリックインストール／アンインストール：install.cmd、uninstall.cmd を実行するだけ
  scripts/            その裏で動く PowerShell スクリプトと VDD の設定
linux/                Linux のワンクリックインストール／アンインストール：install.sh、uninstall.sh を実行するだけ
  scripts/            チェック、ヘルパー、polkit ルール、initramfs フック、.desktop
build.ps1 / Makefile  ビルド
release.ps1           リリースパッケージの作成（手順と内容は RELEASING.md）
```

## 別の方法（不採用）

Microsoft の IddSampleDriver を改造し、EDID をドライバーに直接書き込む方法もあります。すべてのバイトを自分で制御できますが、ドライバーを自分でビルドする必要があります。正式な署名がない場合はテスト署名モードが必要で、テスト署名モードには Secure Boot の無効化が必要なうえ、画面に透かしも表示されます。Virtual Display Driver は署名済みで、カスタム EDID にも対応しているため、こちらを採用しました。

## ライセンス

本プロジェクトは [MIT ライセンス](LICENSE) で公開しています。自由に使用・改変・再配布できますが、保証は一切ありません。同梱のサードパーティソフトウェアは、それぞれのライセンスに従います（「[サードパーティソフトウェア](#サードパーティソフトウェア)」を参照）。

このツールが役に立った場合は、開発への支援（スポンサー）を歓迎します：[GitHub Sponsors](https://github.com/sponsors/The78estHerry)（リポジトリページ上部の「Sponsor」ボタン。審査が通るまでは利用できない場合があります）。

## 商標

- Synergy は Symless Ltd. の商標であり、Synergy のアイコン（ロゴ）は Symless Ltd. に帰属します。本プロジェクトは互換性のためにその名称に言及しているだけで、Symless とは一切関係がなく、承認や後援も受けていません。
- 本プロジェクトは Synergy のアイコンを含んでおらず、配布もしていません。スイッチプログラムに表示される Synergy のアイコンは、ユーザー自身のパソコンにインストールされている Synergy から実行時に読み込んでいます。
- Windows は Microsoft Corporation の商標です。その他の製品名は、それぞれの所有者に帰属します。

## サードパーティソフトウェア

Windows のリリースパッケージには、Virtual Display Driver 公式の `VDD.Control.25.7.23.zip` から取り出した次のファイルを、改変せずに同梱しています。

| ファイル | ソフトウェア | ライセンス |
|---|---|---|
| `scripts\vdd\SignedDrivers\*\VDD\*` | [Virtual Display Driver](https://github.com/VirtualDrivers/Virtual-Display-Driver) | MIT |
| `scripts\vdd\Dependencies\devcon.exe` | DevCon（Microsoft の署名付き、[ソースコード](https://github.com/microsoft/Windows-driver-samples/tree/main/setup/devcon)） | MS-PL |

ライセンスの全文は `windows\scripts\THIRD-PARTY-NOTICES.txt`（パッケージ内では `scripts\vdd\THIRD-PARTY-NOTICES.txt`）にあります。本プロジェクトは、これらのソフトウェアの作者とは一切関係がありません。
