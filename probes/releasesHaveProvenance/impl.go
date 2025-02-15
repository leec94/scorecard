// Copyright 2023 OpenSSF Scorecard Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//nolint:stylecheck
package releasesHaveProvenance

import (
	"embed"
	"fmt"
	"strings"

	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/finding"
	"github.com/ossf/scorecard/v4/probes/internal/utils/uerror"
)

//go:embed *.yml
var fs embed.FS

const (
	Probe           = "releasesHaveProvenance"
	releaseLookBack = 5
)

// ValueType is the type of a value in the finding values.
type ValueType int

const (
	// ValueTypeRelease represents a release name.
	ValueTypeRelease ValueType = iota
	// ValueTypeReleaseAsset represents a release asset name.
	ValueTypeReleaseAsset
)

var provenanceExtensions = []string{".intoto.jsonl"}

func Run(raw *checker.RawResults) ([]finding.Finding, string, error) {
	if raw == nil {
		return nil, "", fmt.Errorf("%w: raw", uerror.ErrNil)
	}

	var findings []finding.Finding

	releases := raw.SignedReleasesResults.Releases

	totalReleases := 0

	for i := range releases {
		release := releases[i]
		if len(release.Assets) == 0 {
			continue
		}
		totalReleases++
		hasProvenance := false
		for j := range release.Assets {
			asset := release.Assets[j]
			for _, suffix := range provenanceExtensions {
				if !strings.HasSuffix(asset.Name, suffix) {
					continue
				}
				// Create Positive Finding
				// with file info
				loc := &finding.Location{
					Type: finding.FileTypeURL,
					Path: asset.URL,
				}
				f, err := finding.NewWith(fs, Probe,
					fmt.Sprintf("provenance for release artifact: %s", asset.Name),
					loc,
					finding.OutcomePositive)
				if err != nil {
					return nil, Probe, fmt.Errorf("create finding: %w", err)
				}
				f.Values = map[string]int{
					release.TagName: int(ValueTypeRelease),
					asset.Name:      int(ValueTypeReleaseAsset),
				}
				findings = append(findings, *f)
				hasProvenance = true
				break
			}
			if hasProvenance {
				break
			}
		}
		if hasProvenance {
			continue
		}

		// Release does not have provenance
		loc := &finding.Location{
			Type: finding.FileTypeURL,
			Path: release.URL,
		}
		f, err := finding.NewWith(fs, Probe,
			fmt.Sprintf("release artifact %s does not have provenance", release.TagName),
			loc,
			finding.OutcomeNegative)
		if err != nil {
			return nil, Probe, fmt.Errorf("create finding: %w", err)
		}
		f.Values = map[string]int{
			release.TagName: int(ValueTypeRelease),
		}
		findings = append(findings, *f)

		if totalReleases >= releaseLookBack {
			break
		}
	}

	if len(findings) == 0 {
		f, err := finding.NewWith(fs, Probe,
			"no GitHub releases found",
			nil,
			finding.OutcomeNotApplicable)
		if err != nil {
			return nil, Probe, fmt.Errorf("create finding: %w", err)
		}
		findings = append(findings, *f)
	}
	return findings, Probe, nil
}
