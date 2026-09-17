package identity

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Locations 经典武侠门派与江湖圣地
var Locations = []string{
	"光明顶", "桃花岛", "黑木崖", "终南山", "绝情谷", "聚贤庄",
	"武当金顶", "少林藏经阁", "华山思过崖", "天山灵鹫宫", "大理万劫谷", "辽东神龙岛",
	"南海侠客岛", "昆仑恶人谷", "关中七侠镇", "同福客栈", "天山缥缈峰", "塞外雁门关",
	"雪山玉笔峰", "峨眉金顶", "极北冰火岛", "绣玉移花宫", "东海蝙蝠岛", "终南活死人墓",
	"无锡杏子林", "终南百花谷", "擂鼓山棋局", "姑苏听香水榭", "太湖曼陀山庄", "姑苏燕子坞",
	"洛阳绿竹巷", "风陵渡口", "昆仑三圣坳", "皖南蝴蝶谷", "塞北白驼山",
}

// Characters 经典武侠名宿与隐世高人
var Characters = []string{
	"张无忌", "郭靖", "黄蓉", "令狐冲", "杨过", "小龙女",
	"乔峰", "虚竹", "段誉", "周伯通", "黄药师", "洪七公",
	"欧阳锋", "一灯大师", "张三丰", "风清扬", "任我行", "东方不败",
	"西门吹雪", "陆小凤", "楚留香", "李寻欢", "花满楼", "谢逊",
	"韦一笑", "金轮法王", "鸠摩智", "慕容复", "游坦之", "丁春秋",
	"岳不群", "林平之", "石破天", "苗人凤", "胡斐", "江小鱼",
	"花无缺", "燕南天", "邀月", "怜星", "司空摘星", "阿飞",
	"楚鹿山", "百晓生", "无崖子", "天山童姥", "李秋水",
}

// IdentityColors 给侠客分配的武侠高对比终端主色调
var IdentityColors = []string{
	"#00f0ff", // 凌冽寒冰青
	"#39ff14", // 碧竹荧光绿
	"#ff007f", // 霓虹胭脂红
	"#ffd700", // 纯金琉璃黄
	"#ff7700", // 烈焰赤阳橙
	"#bf55ec", // 紫霞玄冥紫
	"#00e5ff", // 沧海浮屠蓝
	"#ff4b4b", // 朱砂剑芒赤
	"#e056fd", // 幻影罗刹粉
	"#26de81", // 翠玉竹叶青
}

// GenerateIdentity 随机生成一个武侠江湖身份，如 "【光明顶·张无忌】"
func GenerateIdentity() string {
	locIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(Locations))))
	if err != nil {
		locIdx = big.NewInt(0)
	}
	charIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(Characters))))
	if err != nil {
		charIdx = big.NewInt(0)
	}

	return fmt.Sprintf("【%s·%s】", Locations[locIdx.Int64()], Characters[charIdx.Int64()])
}

// GetIdentityColor 根据身份字符串哈希，为该侠客分配稳定的终端视觉高亮色
func GetIdentityColor(identity string) string {
	if len(identity) == 0 {
		return IdentityColors[0]
	}
	var hash uint32
	for _, r := range identity {
		hash = hash*31 + uint32(r)
	}
	idx := int(hash % uint32(len(IdentityColors)))
	return IdentityColors[idx]
}
