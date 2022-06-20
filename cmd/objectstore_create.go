package cmd

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/briandowns/spinner"
	"github.com/civo/civogo"
	"github.com/civo/cli/config"
	"github.com/civo/cli/utility"
	"github.com/spf13/cobra"
)

var maxObjects, bucketSize int
var waitOS bool

var objectStoreCreateCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"new", "add"},
	Example: "civo objectstore create OBJECTSTORE_NAME_PREFIX --size SIZE",
	Short:   "Create a new Object Store",
	Long:    "Bucket size should be in Gigabytes (GB) and must be a multiple of 500, starting from 500.\n An Object Store name will be generated from the prefix provided.\n",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		utility.EnsureCurrentRegion()

		client, err := config.CivoAPIClient()
		if err != nil {
			utility.Error("Creating the connection to Civo's API failed with %s", err)
			os.Exit(1)
		}

		if regionSet != "" {
			client.Region = regionSet
		}

		if bucketSize == 0 {
			bucketSize = 500
		}
		store, err := client.NewObjectStore(&civogo.CreateObjectStoreRequest{
			Name:       args[0],
			MaxSizeGB:  bucketSize,
			MaxObjects: maxObjects,
			Region:     client.Region,
		})
		if err != nil {
			utility.Error("%s", err)
			os.Exit(1)
		}

		var executionTime string
		if waitOS {
			startTime := utility.StartTime()
			stillCreating := true
			s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
			s.Prefix = fmt.Sprintf("Creating an Object Store with maxSize %s, maxObjects %s called %s... ", store.MaxSize, strconv.Itoa(store.MaxObjects), store.Name)
			s.Start()

			for stillCreating {
				storeCheck, err := client.FindObjectStore(store.ID)
				if err != nil {
					utility.Error("Object Store %s", err)
					os.Exit(1)
				}
				if storeCheck.Status == "ready" {
					stillCreating = false
					s.Stop()
				} else {
					time.Sleep(2 * time.Second)
				}
			}

			executionTime = utility.TrackTime(startTime)
		}

		objectStore, err := client.FindObjectStore(args[0])
		if err != nil {
			utility.Error("ObjectStore %s", err)
			os.Exit(1)
		}

		ow := utility.NewOutputWriterWithMap(map[string]string{"name": objectStore.Name, "id": objectStore.ID, "access_key": objectStore.AccessKeyID})

		switch outputFormat {
		case "json":
			ow.WriteSingleObjectJSON(prettySet)
		case "custom":
			ow.WriteCustomOutput(outputFields)
		default:
			if waitOS {
				fmt.Printf("Created Object Store %s in %s in %s\n", utility.Green(objectStore.Name), utility.Green(client.Region), executionTime)
				fmt.Printf("Created default admin credentials, access key is %s, this will be deleted if the Object Store is deleted. ", utility.Green(objectStore.AccessKeyID))
				fmt.Printf("To access the secret key run: civo objectstore credential secret --access-key=%s\n", utility.Green(objectStore.AccessKeyID))
			} else {
				fmt.Printf("Creating Object Store %s in %s\n", utility.Green(objectStore.Name), utility.Green(client.Region))
				fmt.Printf("To check the status of the Object Store run: civo objectstore show %s\n", utility.Green(objectStore.Name))
			}
		}
	},
}
