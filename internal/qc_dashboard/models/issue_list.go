package models

var IssueList = []IssueListItem{
	{ReasonCode: "SURFACE_DEFECT", ReasonText: "Surface Defect", Category: "visual"},
	{ReasonCode: "DIM_OUT_SPEC", ReasonText: "Dimension Out of Spec", Category: "dimensional"},
	{ReasonCode: "PAINT_DEFECT", ReasonText: "Paint Defect", Category: "visual"},
	{ReasonCode: "MISS_HOLE", ReasonText: "Missing Hole", Category: "visual"},
	{ReasonCode: "MATERIAL_DMG", ReasonText: "Material Damage", Category: "material"},
	{ReasonCode: "OTHER", ReasonText: "Other", Category: "other"},
}
