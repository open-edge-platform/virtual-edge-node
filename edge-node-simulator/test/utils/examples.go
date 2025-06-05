// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"time"

	"github.com/open-edge-platform/infra-core/apiv2/v2/pkg/api/v2"
	os_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/os/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
)

const (
	SafeTimeDelay = 600
	DelayStart1   = 1
	DelayStart5   = 5
	DelayStart10  = 10
	DelayStart20  = 20
	DelayStart50  = 50
	DurationTest  = 10
	schedName     = "schedSingle"
)

var (
	TimeNow = uint64(time.Now().UTC().Unix())

	MmSingleSchedulePastContinuing = pb.SingleSchedule{
		StartSeconds: TimeNow - SafeTimeDelay - DelayStart20,
	}

	MmSingleSchedule0 = pb.SingleSchedule{
		StartSeconds: TimeNow + SafeTimeDelay + DelayStart10,
	}
	MmSingleSchedule1 = pb.SingleSchedule{
		StartSeconds: TimeNow + DelayStart1,
		EndSeconds:   TimeNow + DelayStart5,
	}

	MmRepeatedSchedule1 = []*pb.RepeatedSchedule{
		{
			DurationSeconds: uint32(DurationTest),
			CronMinutes:     "*",
			CronHours:       "*",
			CronDayMonth:    "*",
			CronMonth:       "*",
			CronDayWeek:     "*",
		},
	}
	OSResource = &os_v1.OperatingSystemResource{
		Name:              "for unit testing purposes",
		UpdateSources:     []string{"test entries"},
		ImageUrl:          "Repo URL Test",
		ProfileName:       inv_testing.GenerateRandomProfileName(),
		Sha256:            inv_testing.GenerateRandomSha256(),
		InstalledPackages: "intel-opencl-icd\nintel-level-zero-gpu\nlevel-zero",
		SecurityFeature:   os_v1.SecurityFeature_SECURITY_FEATURE_UNSPECIFIED,
	}

	OSUpdateSources = []string{
		"#ReleaseService\nTypes: deb\nURIs: https://files-rs.edgeorchestration.intel.com\nSuites: 24.03" +
			"\nComponents: main\nSigned-By:\n -----BEGIN PGP PUBLIC KEY BLOCK-----\n .\n " +
			"mQINBGXE3tkBEAD85hzXnrq6rPnOXxwns35NfLaT595jJ3r5J17U/heOymT+K18D\n " +
			"A6ewAwQgyHEWemW87xW6iqzRI4jB5m/fnFvl8wS1JmE8tZMYxLZDav91XfHNzV7J\n " +
			"pgI+5zQ2ojD1yIwmJ6ILo/uPNGYxvdCaUX1LcqELXVRqmg64qEOEMfA6fjfUUocm\n " +
			"bhx9Yf6dLYplJ3sgRTJQ0jY0LdAE8yicPXheGT+vtxWs/mM64KrIafbuGqNiYwC3\n " +
			"e0cHWMPCLVe/lZcPjpaSpx03e0nVno50Xzod7PgVT+qI/l7STS0vT1TQK9IJPE1X\n " +
			"8auCEE0Z/sT+Q/6Zs4LiJnRZqBLoPFbyt7aZstS/zzYtX5qkv8iGaIo3CCxVN74u\n " +
			"Gr4B01H3T55kZ4LE1pzrkB/9w4EDGC2KSyJg2vzqQP6YU8yeArJrcxhHUkNnVmjg\n " +
			"GYeOiIpm+S4X6mD69T8r/ohIdQRggAEAMsiC+Lru6mtesKC8Ju0zdQIZWAiZiI5m\n " +
			"u88UqT/idq/FFSdWb8zMTzE0doTVxZu2ScW99Vw3Bhl82w6lY689mqfHN6HAw3Oj\n " +
			"CXGBd4IooalwjGCg27jNTZ5HiImK1Pi2wnlMdFyCXR4BPwjHMfEr1av3m4U9OkfB\n " +
			"lVPHS35v0/y22e6FENg7kUiucY4ytKbbAMFeVIwVopHOhpDT29dUtfRsZwARAQAB\n " +
			"tAVJbnRlbIkCTgQTAQoAOBYhBNBzdS76jrQWu9oBzLoBs/zr58/PBQJlxN7ZAhsD\n " +
			"BQsJCAcCBhUKCQgLAgQWAgMBAh4BAheAAAoJELoBs/zr58/PboUQAMAP8f2plI1W\n " +
			"Zypc+CszsnRMUqDtwiqA56Q+ZTc6Tdb/P7Isw/lLno3LgL4fkip8Yxmql9zA4aXk\n " +
			"EnNd3mPZcZdP2fogkgOd2gqbmcu604P3kUrlIrrWbSpyH+qmtwfyV09j7xucQ527\n " +
			"+1gXGwVNXcqrmgUWlYTXD+SIeXosmWPvAJgF2PvI1ETTjXvpJryNaaekw1gmRYfs\n " +
			"Jiq6LPGvPkyefcgXRD2lgTWnMRpAfiukIhZro0YLIqj3godF2qsmu3Xb6IhFFHFN\n " +
			"eL9IVqJW/cEsFD21P5sC6FjQjV+Hu2jRTPFVHsTEiF34XC2LNDiVaZWtLIhWXjas\n " +
			"FTwBw2vqGaWRUhAUWzmvfS97XGx5gDMdODNfwGfsFzDLfmuW7gFaT/qkc07KmaYb\n " +
			"QobESazmA51UiEcxOwUZWsVwWM259YIc2TTndkCJf2P9rOXLCmOYbtOZqLcnpE4O\n " +
			"tKkATRwwSP2uOyMmkwRbTwazR5ZMJ1tAO+ewl2guyDcJuk/tboh57AZ40JFRlzz4\n " +
			"dKybtByZ2ntW/sYvXwR818/sUd2PjtRHekBq+bprw2JR2OwPhfAswBs9UzWNiSqd\n " +
			"rA3NksCeuj/j6sSaqpXn123ZtlliZttviM+bvbSps5qJ5TbxHtSwr4H5gYSlHVT/\n " +
			"IwqUfFrYNoQVDejlGkVgyjQYonEqk8eX\n " +
			"=w4R+\n -----END PGP PUBLIC KEY BLOCK-----",

		"Types: deb\nURIs: https://repo.mongodb.org/apt/ubuntu\nSuites: jammy/mongodb-org/7.0" +
			"\nComponents: multiverse\nSigned-By:\n -----BEGIN PGP PUBLIC KEY BLOCK-----\n " +
			"Version: GnuPG v1\n .\n " +
			"mQINBGPILWABEACqeWP/ktugdlWEyk7YTXo3n19+5Om4AlSdIyKv49vAlKtzCfMA\n " +
			"QkZq3mfvjXiKMuLnL2VeElAJQIYcPoqnHf6tJbdrNv4AX2uI1cTsvGW7YS/2WNwJ\n " +
			"C/+vBa4o+yA2CG/MVWZRbtOjkFF/W07yRFtNHAcgdmpIjdWgSnPQr9eIqLuWXIhy\n " +
			"H7EerKsba227Vd/HfvKnAy30Unlsdywy7wi1FupzGJck0TPoOVGmsSpSyIQu9A4Z\n " +
			"uC6TE/NcJHvaN0JuHwM+bQo9oWirGsZ1NCoVqSY8/sasdUc7T9r90MbUcH674YAR\n " +
			"8OKYVBzU0wch4VTFhfHZecKHQnZf+V4dmP9oXnu4fY0/0w3l4jaew7Ind7kPg3yN\n " +
			"hvgAkBK8yRAbSu1NOtHDNiRoHGEQFgct6trVOvCqHbN/VToLNtGk0rhKGOp8kuSF\n " +
			"OJ02PJPxF3/zHGP8n8khCjUJcrilYPqRghZC8ZWnCj6GJVg6WjwLi+hPwNMi8xK6\n " +
			"cjKhRW3eCy5Wcn73PzVBX9f7fSeFDJec+IfS47eNkxunHAOUMXa2+D+1xSWgEfK0\n " +
			"PClfyWPgLIXY2pGQ6v8l3A6P5gJv4o38/E1h1RTcO3H1Z6cgZLIORZHPyAj50SPQ\n " +
			"cjzftEcz56Pl/Cyw3eMYC3qlbABBgsdeb6KB6G5dkNxI4or3MgmxcwfnkwARAQAB\n " +
			"tDdNb25nb0RCIDcuMCBSZWxlYXNlIFNpZ25pbmcgS2V5IDxwYWNrYWdpbmdAbW9u\n " +
			"Z29kYi5jb20+iQI+BBMBAgAoBQJjyC1gAhsDBQkJZgGABgsJCAcDAgYVCAIJCgsE\n " +
			"FgIDAQIeAQIXgAAKCRAWDSa7F4W6OM+eD/sE7KbJyRNWyPCRTqqJXrXvyPqZtbFX\n " +
			"8sio0lQ8ghn4f7lmb7LnFroUsmBeWaYirM8O3b2+iQ9oj4GeR3gbRZsEhFXQfL54\n " +
			"SfrmG9hrWWpJllgPP7Six+jrzcjvkf1TENqw4jRP+cJhuihH1Gfizo9ktwwoN9Yr\n " +
			"m7vgh+focEEmx8dysS38ApLxKlUEfTsE9bYsClgqyY1yrt3v4IpGbf66yfyBHNgY\n " +
			"sObR3sngDRVbap7PwNyREGsuAFfKr/Dr37HfrjY7nsn3vH7hbDpSBh+H7a0b/chS\n " +
			"mM60aaG4biWpvmSC7uxA/t0gz+NQuC4HL+qyNPUxvyIO+TwlaXfCI6ixazyrH+1t\n " +
			"F7Bj5mVsne7oeWjRrSz85jK3Tpn9tj3Fa7PCDA6auAlPK8Upbhuoajev4lIydNd2\n " +
			"70yO0idm/FtpX5a8Ck7KSHDvEnXpN70imayoB4Fs2Kigi2BdZOOdib16o5F/9cx9\n " +
			"piNa7HotHCLTfR6xRmelGEPWKspU1Sm7u2A5vWgjfSab99hiNQ89n+I7BcK1M3R1\n " +
			"w/ckl6qBtcxz4Py+7jYIJL8BYz2tdreWbdzWzjv+XQ8ZgOaMxhL9gtlfyYqeGfnp\n " +
			"hYW8LV7a9pavxV2tLuVjMM+05ut/d38IkTV7OSJgisbSGcmycXIzxsipyXJVGMZt\n " +
			"MFw3quqJhQMRsA==\n =gbRM\n -----END PGP PUBLIC KEY BLOCK---",
	}

	OSInstalledPackages = "net-tools\nmongodb-org"
	OSKernelCmd         = "abcdef"

	Region1Name = "region-12345678"
	Region2Name = "region-23456789"

	Site1Name = "site-12345678"
	Site2Name = "site-12345679"

	Site1Request = api.SiteResource{
		Name: &Site1Name,
	}
	Site2Request = api.SiteResource{
		Name: &Site2Name,
	}

	Region1Request = api.RegionResource{
		Name: &Region1Name,
	}

	Region2Request = api.RegionResource{
		Name: &Region2Name,
	}
)
