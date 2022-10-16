package contentSafe

import (
	"github.com/23233/ggg/sv"
	"github.com/kataras/iris/v12"
)

func RegistryRouters(party iris.Party) {
	party.Post("/hit_text", sv.Run(new(WordValidReq)), HitText)
	party.Post("/hit_image", sv.Run(new(HitImgReq)), HitImg)
}
