package types

type CheckerConfig struct {
	lowPeer      LowPeer      `yaml:"lowPeer"`
	missingBlock MissingBlock `yaml:"missingBlock"`
	heightStuck  HeightStuck  `yaml:"heightStuck"`
	heartbeat    Heartbeat    `yaml:"heartbeat"`
}

type Alert interface {
	getAlertCode() string
	setAlertCode(name string)

	getAlertLevel() string
	setAlertLevel(name string)
}

type LowPeer struct {
	alertCode  string `yaml:"alertCode"`
	alertLevel string `yaml:"level"`
	threshold  string
}

func (l *LowPeer) getAlertCode() string {
	return l.alertCode
}

func (l *LowPeer) setAlertCode(name string) {
	l.alertCode = name
}

func (l *LowPeer) getAlertLevel() string {
	return l.alertLevel
}

func (l *LowPeer) setAlertLevel(name string) {
	l.alertLevel = name
}

type MissingBlock struct {
	alertCode  string `yaml:"alertCode"`
	alertLevel string `yaml:"level"`
}

func (l *MissingBlock) getAlertCode() string {
	return l.alertCode
}

func (l *MissingBlock) setAlertCode(name string) {
	l.alertCode = name
}

func (l *MissingBlock) getAlertLevel() string {
	return l.alertLevel
}

func (l *MissingBlock) setAlertLevel(name string) {
	l.alertLevel = name
}

type HeightStuck struct {
	alertCode  string `yaml:"alertCode"`
	alertLevel string `yaml:"level"`
}

func (l *HeightStuck) getAlertCode() string {
	return l.alertCode
}

func (l *HeightStuck) setAlertCode(name string) {
	l.alertCode = name
}

func (l *HeightStuck) getAlertLevel() string {
	return l.alertLevel
}

func (l *HeightStuck) setAlertLevel(name string) {
	l.alertLevel = name
}

type Heartbeat struct {
	alertCode  string `yaml:"alertCode"`
	alertLevel string `yaml:"level"`
}

func (l *Heartbeat) getAlertCode() string {
	return l.alertCode
}

func (l *Heartbeat) setAlertCode(name string) {
	l.alertCode = name
}

func (l *Heartbeat) getAlertLevel() string {
	return l.alertLevel
}

func (l *Heartbeat) setAlertLevel(name string) {
	l.alertLevel = name
}
