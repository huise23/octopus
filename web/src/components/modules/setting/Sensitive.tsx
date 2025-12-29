'use client';

import { useEffect, useState, useRef } from 'react';
import { useTranslations } from 'next-intl';
import { Shield, Plus, Trash2, Pencil, Lock } from 'lucide-react';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import { useSensitiveRuleList, useCreateSensitiveRule, useUpdateSensitiveRule, useDeleteSensitiveRule, useToggleSensitiveRule, type SensitiveFilterRule } from '@/api/endpoints/sensitive';
import { toast } from '@/components/common/Toast';

export function SettingSensitive() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const { data: rules } = useSensitiveRuleList();
    const setSetting = useSetSetting();
    const createRule = useCreateSensitiveRule();
    const updateRule = useUpdateSensitiveRule();
    const deleteRule = useDeleteSensitiveRule();
    const toggleRule = useToggleSensitiveRule();

    const [globalEnabled, setGlobalEnabled] = useState(true);
    const [showDialog, setShowDialog] = useState(false);
    const [editingRule, setEditingRule] = useState<SensitiveFilterRule | null>(null);
    const [formData, setFormData] = useState({ name: '', pattern: '', replacement: '', priority: 0 });

    const initialGlobalEnabled = useRef(true);

    useEffect(() => {
        if (settings) {
            const setting = settings.find(s => s.key === SettingKey.SensitiveFilterEnabled);
            if (setting) {
                const enabled = setting.value === 'true';
                setGlobalEnabled(enabled);
                initialGlobalEnabled.current = enabled;
            }
        }
    }, [settings]);

    const handleGlobalToggle = (checked: boolean) => {
        setGlobalEnabled(checked);
        setSetting.mutate(
            { key: SettingKey.SensitiveFilterEnabled, value: checked ? 'true' : 'false' },
            { onSuccess: () => { toast.success(t('saved')); initialGlobalEnabled.current = checked; } }
        );
    };

    const handleRuleToggle = (rule: SensitiveFilterRule) => {
        toggleRule.mutate({ id: rule.id, enabled: !rule.enabled });
    };

    const handleAddRule = () => {
        setEditingRule(null);
        setFormData({ name: '', pattern: '', replacement: '[FILTERED]', priority: 0 });
        setShowDialog(true);
    };

    const handleEditRule = (rule: SensitiveFilterRule) => {
        if (rule.built_in) return;
        setEditingRule(rule);
        setFormData({ name: rule.name, pattern: rule.pattern, replacement: rule.replacement, priority: rule.priority });
        setShowDialog(true);
    };

    const handleDeleteRule = (rule: SensitiveFilterRule) => {
        if (rule.built_in) return;
        deleteRule.mutate(rule.id, { onSuccess: () => toast.success(t('sensitive.deleted')) });
    };

    const handleSaveRule = () => {
        if (!formData.name || !formData.pattern) {
            toast.error(t('sensitive.requiredFields'));
            return;
        }
        try {
            new RegExp(formData.pattern);
        } catch {
            toast.error(t('sensitive.invalidPattern'));
            return;
        }

        if (editingRule) {
            updateRule.mutate({ ...editingRule, ...formData }, {
                onSuccess: () => { toast.success(t('saved')); setShowDialog(false); }
            });
        } else {
            createRule.mutate({ ...formData, enabled: true }, {
                onSuccess: () => { toast.success(t('sensitive.created')); setShowDialog(false); }
            });
        }
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 custom-shadow space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Shield className="h-5 w-5" />
                {t('sensitive.title')}
            </h2>

            {/* 全局开关 */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Shield className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('sensitive.globalEnabled')}</span>
                </div>
                <Switch checked={globalEnabled} onCheckedChange={handleGlobalToggle} />
            </div>

            {/* 规则列表 */}
            <div className="space-y-2">
                <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-muted-foreground">{t('sensitive.rules')}</span>
                    <Button variant="outline" size="sm" onClick={handleAddRule} className="rounded-xl">
                        <Plus className="h-4 w-4 mr-1" />{t('sensitive.addRule')}
                    </Button>
                </div>
                <div className="space-y-2 max-h-64 overflow-y-auto">
                    {rules?.map(rule => (
                        <div key={rule.id} className="flex items-center justify-between p-3 rounded-xl bg-muted/50 gap-3">
                            <div className="flex items-center gap-3 flex-1 min-w-0">
                                <Switch checked={rule.enabled} onCheckedChange={() => handleRuleToggle(rule)} disabled={!globalEnabled} />
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm font-medium truncate">{rule.name}</span>
                                        {rule.built_in && <Lock className="h-3 w-3 text-muted-foreground" />}
                                    </div>
                                    <span className="text-xs text-muted-foreground font-mono truncate block">{rule.pattern}</span>
                                </div>
                            </div>
                            {!rule.built_in && (
                                <div className="flex items-center gap-1">
                                    <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => handleEditRule(rule)}>
                                        <Pencil className="h-4 w-4" />
                                    </Button>
                                    <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive" onClick={() => handleDeleteRule(rule)}>
                                        <Trash2 className="h-4 w-4" />
                                    </Button>
                                </div>
                            )}
                        </div>
                    ))}
                </div>
            </div>

            {/* 添加/编辑对话框 */}
            <Dialog open={showDialog} onOpenChange={setShowDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>{editingRule ? t('sensitive.editRule') : t('sensitive.addRule')}</DialogTitle>
                        <DialogDescription>{t('sensitive.ruleDescription')}</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="name">{t('sensitive.ruleName')}</Label>
                            <Input id="name" value={formData.name} onChange={e => setFormData({ ...formData, name: e.target.value })} placeholder="e.g. Custom API Key" />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="pattern">{t('sensitive.pattern')}</Label>
                            <Input id="pattern" value={formData.pattern} onChange={e => setFormData({ ...formData, pattern: e.target.value })} placeholder="e.g. my-secret-[a-z]+" className="font-mono" />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="replacement">{t('sensitive.replacement')}</Label>
                            <Input id="replacement" value={formData.replacement} onChange={e => setFormData({ ...formData, replacement: e.target.value })} placeholder="[FILTERED]" />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="priority">{t('sensitive.priority')}</Label>
                            <Input id="priority" type="number" value={formData.priority} onChange={e => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })} />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setShowDialog(false)}>{t('sensitive.cancel')}</Button>
                        <Button onClick={handleSaveRule}>{t('sensitive.save')}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
