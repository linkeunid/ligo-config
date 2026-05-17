package ligo_config

import "testing"

func TestModuleWith_PublishesService(t *testing.T) {
	svc := newServiceWith(map[string]string{"KEY": "value"})
	m := ModuleWith(svc)
	if m.Name != ModuleName {
		t.Errorf("Module name = %q, want %q", m.Name, ModuleName)
	}
}

func TestModuleWith_NilSvcSafe(t *testing.T) {
	m := ModuleWith(nil)
	if m.Name != ModuleName {
		t.Error("ModuleWith(nil) should still return a valid module")
	}
}
