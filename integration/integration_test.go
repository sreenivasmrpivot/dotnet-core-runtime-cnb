package integration

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	bp string
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	root, err := dagger.FindBPRoot()
	Expect(err).ToNot(HaveOccurred())
	bp, err = dagger.PackageBuildpack(root)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		dagger.DeleteBuildpack(bp)
	}()

	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app *dagger.App
		err error
	)
	it.Before(func() {
		Expect = NewWithT(t).Expect
	})
	it.After(func() {
		if app != nil {
			app.Destroy()
		}
	})

	it("runs a simple framework-dependent deployment with a framework-dependent executable", func() {
		app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app"), bp)
		Expect(err).ToNot(HaveOccurred())
		app.Memory = "128m"
		Expect(app.StartWithCommand("./source_code")).To(Succeed())

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello world!"))
	})

	it("runs a simple framework-dependent deployment with a framework-dependent executable that has a buildpack.yml in it", func() {
		majorMinor := "2.2"
		version, err := getLowestRuntimeVersionInMajorMinor(majorMinor)
		Expect(err).ToNot(HaveOccurred())
		bpYml := fmt.Sprintf(`---
dotnet-framework:
  version: "%s"
`, version)

		bpYmlPath := filepath.Join("testdata", "simple_app_with_buildpack_yml", "buildpack.yml")
		Expect(ioutil.WriteFile(bpYmlPath, []byte(bpYml), 0644)).To(Succeed())

		app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app_with_buildpack_yml"), bp)
		Expect(err).ToNot(HaveOccurred())
		app.Memory = "128m"
		Expect(app.StartWithCommand("./source_code")).To(Succeed())

		Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-runtime.%s", version)))

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello world!"))
	})

	// Todo - Copied from V2 to be fixed for V3
	it("runs a simple framework-dependent deployment with a self_contained_2.1 executable", func() {
		// app, err = dagger.PackBuild(filepath.Join("testdata", "self_contained_2.1"), bp)
		// Expect(err).ToNot(HaveOccurred())
		// app.Memory = "128m"
		// Expect(app.StartWithCommand("./self_contained_2.1")).To(Succeed())

		// body, _, err := app.HTTPGet("/")
		// Expect(err).NotTo(HaveOccurred())
		// Expect(body).To(ContainSubstring("Hello world!"))
	})
}

func getLowestRuntimeVersionInMajorMinor(majorMinor string) (string, error) {
	type buildpackTomlVersion struct {
		Metadata struct {
			Dependencies []struct {
				Version string `toml:"version"`
			} `toml:"dependencies"`
		} `toml:"metadata"`
	}

	bpToml := buildpackTomlVersion{}
	output, err := ioutil.ReadFile(filepath.Join("..", "buildpack.toml"))
	if err != nil {
		return "", err
	}

	majorMinorConstraint, err := semver.NewConstraint(fmt.Sprintf("%s.*", majorMinor))
	if err != nil {
		return "", err
	}

	lowestVersion, err := semver.NewVersion(fmt.Sprintf("%s.99", majorMinor))
	if err != nil {
		return "", err
	}

	_, err = toml.Decode(string(output), &bpToml)
	if err != nil {
		return "", err
	}

	for _, dep := range bpToml.Metadata.Dependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			return "", err
		}
		if majorMinorConstraint.Check(depVersion){
			if depVersion.LessThan(lowestVersion){
				lowestVersion = depVersion
			}
		}
	}

	return lowestVersion.String(), nil
}
