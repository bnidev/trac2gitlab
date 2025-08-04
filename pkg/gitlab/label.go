package gitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Color struct {
	Name     string
	HexValue string
}

var Colors = struct {
	MagentaPink    Color
	Crimson        Color
	RoseRed        Color
	DarkCoral      Color
	CarrotOrange   Color
	TitaniumYellow Color
	GreenCyan      Color
	DarkSeaGreen   Color
	BlueGray       Color
	Lavender       Color
	DarkViolet     Color
	DeepViolet     Color
	CharcoalGrey   Color
	Gray           Color
}{
	MagentaPink:    Color{Name: "Magenta-pink", HexValue: "#cc338b"},
	Crimson:        Color{Name: "Crimson", HexValue: "#dc143c"},
	RoseRed:        Color{Name: "Rose red", HexValue: "#c21e56"},
	DarkCoral:      Color{Name: "Dark coral", HexValue: "#cd5b45"},
	CarrotOrange:   Color{Name: "Carrot orange", HexValue: "#ed9121"},
	TitaniumYellow: Color{Name: "Titanium yellow", HexValue: "#eee600"},
	GreenCyan:      Color{Name: "Green-cyan", HexValue: "#009966"},
	DarkSeaGreen:   Color{Name: "Dark sea green", HexValue: "#8fbc8f"},
	BlueGray:       Color{Name: "Blue-gray", HexValue: "#6699cc"},
	Lavender:       Color{Name: "Lavender", HexValue: "#e6e6fa"},
	DarkViolet:     Color{Name: "Dark violet", HexValue: "#9400d3"},
	DeepViolet:     Color{Name: "Deep violet", HexValue: "#330066"},
	CharcoalGrey:   Color{Name: "Charcoal grey", HexValue: "#36454f"},
	Gray:           Color{Name: "Gray", HexValue: "#808080"},
}

type Label = gitlab.Label

func (c *Client) GetProjectLabels(projectID int) ([]*Label, error) {
	labels, _, err := c.git.Labels.ListLabels(projectID, &gitlab.ListLabelsOptions{})
	if err != nil {
		return nil, err
	}

	return labels, nil
}

func (c *Client) CreateLabel(projectID int, opts *gitlab.CreateLabelOptions) (*Label, error) {
	if opts.Color == nil {
		opts.Color = &Colors.Gray.HexValue
	}
	label, _, err := c.git.Labels.CreateLabel(projectID, opts)
	if err != nil {
		return nil, err
	}

	return label, nil
}

func (c* Client) GetLabelByID(projectID int, labelID int) (*Label, error) {
	label, _, err := c.git.Labels.GetLabel(projectID, labelID)
	if err != nil {
		return nil, err
	}

	return label, nil
}

func (c *Client) GetLabelbyName(projectID int, name string) (*Label, error) {
	labels, _, err := c.git.Labels.ListLabels(projectID, &gitlab.ListLabelsOptions{Search: &name})
	if err != nil {
		return nil, err
	}

	if len(labels) == 0 {
		return nil, nil
	}

	for _, label := range labels {
		if label.Name == name {
			return label, nil
		}
	}

	return nil, nil
}

func (c *Client) UpdateLabel(projectID int, labelID int, opts *gitlab.UpdateLabelOptions) (*Label, error) {
	label, _, err := c.git.Labels.UpdateLabel(projectID, labelID, opts)
	if err != nil {
		return nil, err
	}

	return label, nil
}
