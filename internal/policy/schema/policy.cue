// AgentGuard FS のポリシースキーマ（DESIGN §7）。バイナリに埋め込み、外部から差し替えさせない。
package schema

import "list"

#Action: "read" | "write" | "delete" | "list"
#Effect: "allow" | "deny"

#Rule: close({
	// 詳細な検証（".." の拒否や正規化）は Go 側の pathutil でも必ず行う。
	path!: string & !="" & !~"^/" & !~"[\\\\:\\x00]"
	actions!: [#Action, ...#Action] & list.UniqueItems()
	effect!:   #Effect
	recursive: *false | bool
})

#Limits: close({
	maxReadBytes!:  int & >=1 & <=67108864
	maxWriteBytes!: int & >=1 & <=67108864
	maxListEntries: *1000 | (int & >=1 & <=100000)
})

// #Test は、ポリシーを書く人が期待する判定。読み込み時に Go の判定器で実行する。
#Test: close({
	// 要求のパスと同じく、グロブではなく具体的なパスを書く。
	path!:   string & !=""
	action!: #Action
	want!:   #Effect
})

#Policy: close({
	version!: 1
	rules!: {
		[string]: #Rule
	}
	limits!: #Limits
	tests?: [...#Test]
})
