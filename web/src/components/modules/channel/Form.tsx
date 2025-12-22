import { AutoGroupType, ChannelType, useFetchModel } from '@/api/endpoints/channel';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { ModelMultiSelect } from '@/components/ui/model-multi-select';
import { useTranslations } from 'next-intl';
import { useState, useRef, useEffect, useMemo, useCallback } from 'react';
import { RefreshCw, X, Plus } from 'lucide-react';

export interface ChannelFormData {
    name: string;
    type: ChannelType;
    base_url: string;
    key: string;
    model: string;
    custom_model: string;
    enabled: boolean;
    proxy: boolean;
    auto_sync: boolean;
    auto_group: AutoGroupType;
}

export interface ChannelFormProps {
    formData: ChannelFormData;
    onFormDataChange: (data: ChannelFormData) => void;
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
    isPending: boolean;
    submitText: string;
    pendingText: string;
    onCancel?: () => void;
    cancelText?: string;
    idPrefix?: string;
}

export function ChannelForm({
    formData,
    onFormDataChange,
    onSubmit,
    isPending,
    submitText,
    pendingText,
    onCancel,
    cancelText,
    idPrefix = 'channel',
}: ChannelFormProps) {
    const t = useTranslations('channel.form');

    // 合并所有模型为一个数组
    const allModels = useMemo(() => {
        const auto = formData.model ? formData.model.split(',').filter(m => m.trim()) : [];
        const custom = formData.custom_model ? formData.custom_model.split(',').filter(m => m.trim()) : [];
        return [...new Set([...auto, ...custom])]; // 去重
    }, [formData.model, formData.custom_model]);

    const fetchModel = useFetchModel();

    // 处理模型变更
    const handleModelsChange = useCallback((models: string[]) => {
        // 将模型分为自动获取的和自定义的
        const fetchedModels = fetchModel.data || [];
        const auto = models.filter(model => fetchedModels.includes(model));
        const custom = models.filter(model => !fetchedModels.includes(model));

        onFormDataChange({
            ...formData,
            model: auto.join(','),
            custom_model: custom.join(',')
        });
    }, [formData, onFormDataChange, fetchModel.data]);

    const handleRefreshModels = async () => {
        if (!formData.base_url || !formData.key) return;
        fetchModel.mutate(
            {
                name: formData.name,
                type: formData.type,
                base_url: formData.base_url,
                key: formData.key,
                model: formData.model,
                enabled: formData.enabled,
                proxy: formData.proxy,
            }
            // 刷新成功后，模型会自动显示在下拉框中（通过 availableModels={fetchModel.data || []}）
            // 不再自动选中所有模型，用户需要手动选择
        );
    };

  
    return (
        <form onSubmit={onSubmit} className="space-y-5 px-1">
            <div className="space-y-2">
                <label htmlFor={`${idPrefix}-name`} className="text-sm font-medium text-card-foreground">
                    {t('name')}
                </label>
                <Input
                    className='rounded-xl'
                    id={`${idPrefix}-name`}
                    type="text"
                    value={formData.name}
                    onChange={(event) => onFormDataChange({ ...formData, name: event.target.value })}
                    required
                />
            </div>

            <div className="space-y-2">
                <label htmlFor={`${idPrefix}-type`} className="text-sm font-medium text-card-foreground">
                    {t('type')}
                </label>
                <Select
                    value={String(formData.type)}
                    onValueChange={(value) => onFormDataChange({ ...formData, type: Number(value) as ChannelType })}
                >
                    <SelectTrigger id={`${idPrefix}-type`} className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent className='rounded-xl'>
                        <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIChat)}>{t('typeOpenAIChat')}</SelectItem>
                        <SelectItem className='rounded-xl' value={String(ChannelType.OpenAIResponse)}>{t('typeOpenAIResponse')}</SelectItem>
                        <SelectItem className='rounded-xl' value={String(ChannelType.Anthropic)}>{t('typeAnthropic')}</SelectItem>
                        <SelectItem className='rounded-xl' value={String(ChannelType.Gemini)}>{t('typeGemini')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>

            <div className="space-y-2">
                <label htmlFor={`${idPrefix}-base`} className="text-sm font-medium text-card-foreground">
                    {t('baseUrl')}
                </label>
                <Input
                    id={`${idPrefix}-base`}
                    type="url"
                    value={formData.base_url}
                    onChange={(event) => onFormDataChange({ ...formData, base_url: event.target.value })}
                    required
                    className='rounded-xl'
                />
            </div>

            <div className="space-y-2">
                <label htmlFor={`${idPrefix}-key`} className="text-sm font-medium text-card-foreground">
                    {t('apiKey')}
                </label>
                <Input
                    id={`${idPrefix}-key`}
                    type="text"
                    value={formData.key}
                    onChange={(event) => onFormDataChange({ ...formData, key: event.target.value })}
                    required
                    className='rounded-xl'
                />
            </div>

            <div className="space-y-3">
                <div className="flex items-center justify-between">
                    <label className="text-sm font-medium text-card-foreground">{t('model')}</label>
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={handleRefreshModels}
                        disabled={!formData.base_url || !formData.key || fetchModel.isPending}
                        className="h-6 px-2 text-xs text-muted-foreground/50 hover:text-muted-foreground hover:bg-transparent"
                    >
                        <RefreshCw className={`h-3 w-3 mr-1 ${fetchModel.isPending ? 'animate-spin' : ''}`} />
                        {t('modelRefresh')}
                    </Button>
                </div>

                <ModelMultiSelect
                    selectedModels={allModels}
                    onModelsChange={handleModelsChange}
                    placeholder={t('modelCustomPlaceholder') || "选择或添加模型..."}
                    disabled={false}
                    maxDisplayItems={8}
                    availableModels={fetchModel.data || []}
                    isLoading={fetchModel.isPending}
                />

                <input type="hidden" name="model" value={formData.model} />
                <input type="hidden" name="custom_model" value={formData.custom_model} />
            </div>

            <div className="space-y-2">
                <label htmlFor={`${idPrefix}-auto-group`} className="text-sm font-medium text-card-foreground">
                    {t('autoGroup')}
                </label>
                <Select
                    value={String(formData.auto_group)}
                    onValueChange={(value) => onFormDataChange({ ...formData, auto_group: Number(value) as AutoGroupType })}
                >
                    <SelectTrigger id={`${idPrefix}-auto-group`} className="rounded-xl w-full border border-border px-4 py-2 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent className='rounded-xl'>
                        <SelectItem className='rounded-xl' value={String(AutoGroupType.None)}>{t('autoGroupNone')}</SelectItem>
                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Fuzzy)}>{t('autoGroupFuzzy')}</SelectItem>
                        <SelectItem className='rounded-xl' value={String(AutoGroupType.Exact)}>{t('autoGroupExact')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
                <label className="flex items-center gap-2">
                    <Switch
                        checked={formData.enabled}
                        onCheckedChange={(checked) => onFormDataChange({ ...formData, enabled: checked })}
                    />
                    <span className="text-sm text-card-foreground">{t('enabled')}</span>
                </label>
                <label className="flex items-center gap-2">
                    <Switch
                        checked={formData.proxy}
                        onCheckedChange={(checked) => onFormDataChange({ ...formData, proxy: checked })}
                    />
                    <span className="text-sm text-card-foreground">{t('proxy')}</span>
                </label>
                <label className="flex items-center gap-2">
                    <Switch
                        checked={formData.auto_sync}
                        onCheckedChange={(checked) => onFormDataChange({ ...formData, auto_sync: checked })}
                    />
                    <span className="text-sm text-card-foreground">{t('autoSync')}</span>
                </label>
            </div>

            <div className={`flex flex-col gap-3 pt-2 ${onCancel ? 'sm:flex-row' : ''}`}>
                {onCancel && cancelText && (
                    <Button
                        type="button"
                        variant="secondary"
                        onClick={onCancel}
                        className="w-full sm:flex-1 rounded-2xl h-12"
                    >
                        {cancelText}
                    </Button>
                )}
                <Button
                    type="submit"
                    disabled={isPending}
                    className="w-full sm:flex-1 rounded-2xl h-12"
                >
                    {isPending ? pendingText : submitText}
                </Button>
            </div>
        </form>
    );
}