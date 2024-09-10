package resource

import (
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
)

type OriginAccessControl struct {
	*AwsResource
}

type OriginAccessControlInputs struct {
	Name string `json:"name"`
}

type OriginAccessControlOutputs struct {
}

func (r *OriginAccessControl) Create(input *OriginAccessControlInputs, output *CreateResult[OriginAccessControlOutputs]) error {
	slog.Info("creating origin access control")
	*output = CreateResult[OriginAccessControlOutputs]{
		ID:   *aws.String("unavailable"),
		Outs: OriginAccessControlOutputs{},
	}
	return nil
}

func (r *OriginAccessControl) Delete(input *DeleteInput[OriginAccessControlOutputs], output *int) error {
	cfg, err := r.config()
	if err != nil {
		return err
	}
	cf := cloudfront.NewFromConfig(cfg)
	resp, err := cf.GetOriginAccessControl(r.context, &cloudfront.GetOriginAccessControlInput{
		Id: aws.String(input.ID),
	})
	if err != nil {
		return err
	}
	_, err = cf.DeleteOriginAccessControl(r.context, &cloudfront.DeleteOriginAccessControlInput{
		Id:      aws.String(input.ID),
		IfMatch: resp.ETag,
	})
	if err != nil {
		return err
	}
	return nil
}
