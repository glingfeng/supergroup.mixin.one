package interceptors

import (
	"bytes"
	"context"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/MixinNetwork/supergroup.mixin.one/session"
	pp "google.golang.org/genproto/googleapis/cloud/vision/v1"
)

func CheckSex(ctx context.Context, data []byte) bool {
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		session.Logger(ctx).Errorf("CheckSex NewImageAnnotatorClient ERROR: %+v", err)
		return true
	}
	image, err := vision.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		session.Logger(ctx).Errorf("CheckSex NewImageFromReader ERROR: %+v", err)
		return true
	}
	safe, err := client.DetectSafeSearch(ctx, image, nil)
	if err != nil {
		session.Logger(ctx).Errorf("CheckSex DetectSafeSearch ERROR: %+v", err)
		return true
	}
	session.Logger(ctx).Infof("CheckSex DetectSafeSearch Adult: %s", safe.Adult)
	if safe.Adult >= pp.Likelihood_LIKELY {
		return true
	}
	return false
}
