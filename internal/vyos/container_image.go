package vyos

import (
	"context"
	"fmt"
	"sync"

	"github.com/foltik/vyos-client-go/client"
)

type VyosContainerImages struct {
	apiClient *client.Client
	mutex     sync.Mutex
}

func NewContainerImages(apiClient *client.Client) *VyosContainerImages {
	config := &VyosContainerImages{
		apiClient: apiClient,
	}
	return config
}

func (vc *VyosContainerImages) ShowAll(ctx context.Context) ([]client.ContainerImage, error) {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()
	return vc.apiClient.ContainerImages.Show(ctx)
}

func (vc *VyosContainerImages) Show(ctx context.Context, name string) (*client.ContainerImage, error) {
	images, err := vc.ShowAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		if fmt.Sprintf("%s:%s", image.Name, image.Tag) == name {
			return &image, nil
		}
	}
	return nil, nil
}

func (vc *VyosContainerImages) Add(ctx context.Context, name string) error {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()
	return vc.apiClient.ContainerImages.Add(ctx, name)
}

func (vc *VyosContainerImages) Delete(ctx context.Context, name string) error {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()
	return vc.apiClient.ContainerImages.Delete(ctx, name)
}
