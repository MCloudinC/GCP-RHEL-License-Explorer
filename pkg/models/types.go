package models

type Project struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type Instance struct {
    Name         string `json:"name"`
    Zone         string `json:"zone"`
    Status       string `json:"status"`
    MachineType  string `json:"machineType"`
    NetworkInterfaces []NetworkInterface `json:"networkInterfaces"`
}

type NetworkInterface struct {
    Name    string `json:"name"`
    Network string `json:"network"`
    IP      string `json:"networkIP"`
}