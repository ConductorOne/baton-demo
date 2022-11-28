package client

type database struct {
	Users    []*User
	Groups   []*Group
	Roles    []*Role
	Projects []*Project
}

func generateDB() *database {
	db := &database{}

	db.Users = []*User{
		{
			Id:    "2IC0Wn5oRQqVVn3COFl1O1zSzV6",
			Name:  "Alice",
			Email: "alice@example.com",
		},
		{
			Id:    "2IC0WoNfqUPT7mgO4FOaViIxBrR",
			Name:  "Bob",
			Email: "bob@example.com",
		},
		{
			Id:    "2IC0Wo34fcTerFEgWmyffXmfrW8",
			Name:  "Carol",
			Email: "carol@example.com",
		},
		{
			Id:    "2IC0Wn7DaxV1xqDpdg7jJRiPtCp",
			Name:  "Dan",
			Email: "dan@example.com",
		},
		{
			Id:    "2IC0WoaHVvl2GIQppXQH0flK1yJ",
			Name:  "Frank",
			Email: "frank@example.com",
		},
	}

	db.Groups = []*Group{
		{
			Id:   "2IC0WmAPkihbFdZhEPsch5N5WNO",
			Name: "Engineers",
			Admins: []string{
				"2IC0Wo34fcTerFEgWmyffXmfrW8", // Carol
			},
			Members: []string{
				"2IC0Wn5oRQqVVn3COFl1O1zSzV6", // Alice
				"2IC0WoNfqUPT7mgO4FOaViIxBrR", // Bob
			},
		},
		{
			Id:   "2IC0WjepYDBsRp6b7cqrumGsVGt",
			Name: "Sales",
			Admins: []string{
				"2IC0WoaHVvl2GIQppXQH0flK1yJ", // Frank
			},
			Members: []string{
				"2IC0Wn7DaxV1xqDpdg7jJRiPtCp", // Dan
			},
		},
	}

	db.Roles = []*Role{
		{
			Id:   "2IC0WmaHecJdzo5jYnQiTh2BVlB",
			Name: "Editor",
			DirectAssignments: []string{
				"2IC0WoaHVvl2GIQppXQH0flK1yJ", // Frank
			},
			GroupAssignments: []string{
				"2IC0WmAPkihbFdZhEPsch5N5WNO", // Engineers
			},
		},
		{
			Id:                "2IC0WkRTFmsXH4P9TjiQnd29XMT",
			Name:              "Reader",
			DirectAssignments: nil, // No direct assignments
			GroupAssignments: []string{
				"2IC0WmAPkihbFdZhEPsch5N5WNO", // Engineers
				"2IC0WjepYDBsRp6b7cqrumGsVGt", // Sales
			},
		},
	}

	db.Projects = []*Project{
		{
			Id:    "2IC0WqENS0dCRHiJ0YvPAidl0D5",
			Name:  "Product X",
			Owner: "2IC0WoNfqUPT7mgO4FOaViIxBrR", // Bob
			GroupAssignments: []string{
				"2IC0WmAPkihbFdZhEPsch5N5WNO", // Engineers
				"2IC0WjepYDBsRp6b7cqrumGsVGt", // Sales
			},
		},
		{
			Id:    "2IC11NXgAkNrKRk9nukbPRRKMhI",
			Name:  "Sales",
			Owner: "2IC0WoaHVvl2GIQppXQH0flK1yJ", // Frank
			GroupAssignments: []string{
				"2IC0WjepYDBsRp6b7cqrumGsVGt", // Sales
			},
		},
	}

	return db
}
