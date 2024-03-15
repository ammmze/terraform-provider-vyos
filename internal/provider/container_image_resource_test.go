package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccContainerImageResourceSimpleContainerImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccContainerImageResourceConfig("docker.io/library/alpine:3.18.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vyos_container_image.test", "id", "docker.io/library/alpine:3.18.0"),
					resource.TestCheckResourceAttr("vyos_container_image.test", "name", "docker.io/library/alpine:3.18.0"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "vyos_container_image.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Update and Read testing
			{
				Config: testAccContainerImageResourceConfig("docker.io/library/alpine:3.18.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("vyos_container_image.test", "name", "docker.io/library/alpine:3.18.0"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccContainerImageResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "vyos_container_image" "test" {
  name = %[1]q
}
`, name)
}
