'use client';

import { useMemo, useState, useCallback, type FormEvent, useEffect } from 'react';
import { Plus, Layers, Sparkles } from 'lucide-react';
import { Reorder } from 'motion/react';
import {
    MorphingDialogClose,
    MorphingDialogTitle,
    MorphingDialogDescription,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { useCreateGroup, useUpdateGroup, type Group, type GroupItem, type GroupMode, type GroupKeyword, type GroupMatchMode } from '@/api/endpoints/group';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Field, FieldLabel, FieldGroup } from '@/components/ui/field';
import { cn } from '@/lib/utils';
import { MemberItem, AddMemberRow, type SelectedMember } from './components';
import { matchesGroupName, memberKey, normalizeKey, buildChannelNameByModelKey, modelChannelKey } from './utils';
import { KeywordConfig } from './KeywordConfig';

function MembersSection({
    members,
    onReorder,
    onRemove,
    onWeightChange,
    onAdd,
    onAutoAdd,
    removingIds,
    emptyText,
    showWeight,
    autoAddDisabled,
}: {
    members: SelectedMember[];
    onReorder: (members: SelectedMember[]) => void;
    onRemove: (id: string) => void;
    onWeightChange: (id: string, weight: number) => void;
    onAdd: (channel: LLMChannel) => void;
    onAutoAdd: () => void;
    removingIds: Set<string>;
    emptyText: string;
    showWeight: boolean;
    autoAddDisabled: boolean;
}) {
    const t = useTranslations('group');
    const [isAdding, setIsAdding] = useState(false);
    const showEmpty = members.filter((m) => !removingIds.has(m.id)).length === 0 && !isAdding;

    const handleConfirmAdd = useCallback((channel: LLMChannel) => {
        onAdd(channel);
        setIsAdding(false);
    }, [onAdd]);

    const handleCancelAdd = useCallback(() => {
        setIsAdding(false);
    }, []);

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 overflow-hidden">
            {/* 标题栏 */}
            <div className="flex items-center justify-between px-3 py-2 border-b border-border/30 bg-muted/50">
                <span className="text-sm font-medium text-foreground">
                    {t('form.items')}
                    {members.length > 0 && (
                        <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                            ({members.length})
                        </span>
                    )}
                </span>
                <div className="flex items-center gap-2">
                    <button
                        type="button"
                        onClick={onAutoAdd}
                        className={cn(
                            'flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium transition-colors',
                            autoAddDisabled
                                ? 'text-muted-foreground/50 cursor-not-allowed'
                                : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                        )}
                        disabled={autoAddDisabled}
                        title={t('form.autoAdd')}
                    >
                        <Sparkles className="size-3.5" />
                        <span>{t('form.autoAdd')}</span>
                    </button>
                    <button
                        type="button"
                        onClick={() => setIsAdding(true)}
                        className={cn(
                            'flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium transition-colors',
                            isAdding
                                ? 'bg-primary/10 text-primary cursor-default'
                                : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                        )}
                        disabled={isAdding}
                    >
                        <Plus className="size-3.5" />
                        <span>{t('form.addItem')}</span>
                    </button>
                </div>
            </div>

            {/* 内容区域 */}
            <div className="relative h-96">
                {/* 空状态 */}
                <div
                    className={cn(
                        'absolute inset-0 flex flex-col items-center justify-center gap-2 text-muted-foreground',
                        'transition-opacity duration-200 ease-out',
                        showEmpty ? 'opacity-100' : 'opacity-0 pointer-events-none'
                    )}
                >
                    <Layers className="size-10 opacity-40" />
                    <span className="text-sm">{emptyText}</span>
                </div>

                {/* 列表区域 */}
                <div
                    className={cn(
                        'h-full overflow-y-auto transition-opacity duration-200',
                        showEmpty ? 'opacity-0' : 'opacity-100'
                    )}
                >
                    <div className="p-2 flex flex-col gap-1.5">
                        {/* 已添加的成员列表 */}
                        {members.length > 0 && (
                            <Reorder.Group
                                axis="y"
                                values={members}
                                onReorder={onReorder}
                                className="flex flex-col gap-1.5"
                            >
                                {members.map((member, index) => (
                                    <MemberItem
                                        key={member.id}
                                        member={member}
                                        onRemove={onRemove}
                                        onWeightChange={onWeightChange}
                                        isRemoving={removingIds.has(member.id)}
                                        index={index}
                                        showWeight={showWeight}
                                    />
                                ))}
                            </Reorder.Group>
                        )}

                        {/* 新增成员项 - 放在列表末尾 */}
                        {isAdding && (
                            <AddMemberRow
                                index={members.length}
                                selectedMembers={members}
                                onConfirm={handleConfirmAdd}
                                onCancel={handleCancelAdd}
                            />
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}

export function CreateDialogContent({ editGroup }: { editGroup?: Group }) {
    const { setIsOpen } = useMorphingDialog();
    const createGroup = useCreateGroup();
    const updateGroup = useUpdateGroup();
    const t = useTranslations('group');
    const { data: modelChannels = [] } = useModelChannelList();

    // 是否为编辑模式
    const isEdit = !!editGroup;

    const [groupName, setGroupName] = useState('');
    const [mode, setMode] = useState<GroupMode>(1);
    const [selectedMembers, setSelectedMembers] = useState<SelectedMember[]>([]);
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());
    const [keywords, setKeywords] = useState<GroupKeyword[]>([]);
    const [matchMode, setMatchMode] = useState<GroupMatchMode>(0);

    // 编辑模式初始化
    useEffect(() => {
        if (isEdit && editGroup) {
            setGroupName(editGroup.name);
            setMode(editGroup.mode);
            setMatchMode(editGroup.match_mode ?? 0);

            // 解析关键字
            try {
                if (editGroup.keywords) {
                    const parsedKeywords = JSON.parse(editGroup.keywords) as GroupKeyword[];
                    setKeywords(Array.isArray(parsedKeywords) ? parsedKeywords : []);
                } else {
                    setKeywords([]);
                }
            } catch (error) {
                console.error('解析关键字失败:', error);
                setKeywords([]);
            }

            // 初始化成员列表
            if (editGroup.items && editGroup.items.length > 0) {
                const channelNameByKey = buildChannelNameByModelKey(modelChannels);
                const members: SelectedMember[] = editGroup.items
                    .sort((a, b) => a.priority - b.priority)
                    .map((item) => ({
                        id: `${item.channel_id}-${item.model_name}-${item.id || 0}`,
                        name: item.model_name,
                        channel_id: item.channel_id,
                        channel_name: channelNameByKey.get(modelChannelKey(item.channel_id, item.model_name)) ?? `Channel ${item.channel_id}`,
                        item_id: item.id,  // 添加item_id，标识这是已存在的成员
                        weight: item.weight,
                    }));
                setSelectedMembers(members);
            }
        }
    }, [isEdit, editGroup, modelChannels]);

    const groupKey = useMemo(() => normalizeKey(groupName), [groupName]);

    const matchedModelChannels = useMemo(() => {
        if (!groupKey) return [];
        return modelChannels.filter((mc) => matchesGroupName(mc.name, groupKey));
    }, [groupKey, modelChannels]);

    const handleAddMember = useCallback((channel: LLMChannel) => {
        const key = memberKey(channel);
        setSelectedMembers((prev) => {
            if (prev.some((m) => m.id === key)) return prev;
            return [...prev, { ...channel, id: key, weight: 1 }];
        });
    }, []);

    const autoAddDisabled = useMemo(() => {
        if (!groupKey) return true;
        const existing = new Set(selectedMembers.map((m) => m.id));
        return matchedModelChannels.length === 0 || !matchedModelChannels.some((mc) => !existing.has(memberKey(mc)));
    }, [groupKey, matchedModelChannels, selectedMembers]);

    const handleAutoAdd = useCallback(() => {
        if (!groupKey) return;
        if (matchedModelChannels.length === 0) return;

        setSelectedMembers((prev) => {
            const existing = new Set(prev.map((m) => m.id));
            const toAdd: SelectedMember[] = [];
            matchedModelChannels.forEach((mc) => {
                const k = memberKey(mc);
                if (!existing.has(k)) {
                    existing.add(k);
                    toAdd.push({ ...mc, id: k, weight: 1 });
                }
            });
            return toAdd.length ? [...prev, ...toAdd] : prev;
        });
    }, [groupKey, matchedModelChannels]);

    const handleWeightChange = useCallback((id: string, weight: number) => {
        setSelectedMembers((prev) => prev.map((m) => m.id === id ? { ...m, weight } : m));
    }, []);

    const handleRemoveMember = useCallback((id: string) => {
        setRemovingIds((prev) => new Set(prev).add(id));
        setTimeout(() => {
            setSelectedMembers((prev) => prev.filter((m) => m.id !== id));
            setRemovingIds((prev) => { const n = new Set(prev); n.delete(id); return n; });
        }, 200);
    }, []);

    const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();

        // 准备关键字数据
        const keywordsJson = keywords.length > 0 ? JSON.stringify(keywords) : '';

        if (isEdit && editGroup) {
            // 编辑模式
            const updates: any = { id: editGroup.id! };

            // 检查名称变更
            if (groupName !== editGroup.name) {
                updates.name = groupName;
            }

            // 检查模式变更
            if (mode !== editGroup.mode) {
                updates.mode = mode;
            }

            // 检查关键字变更
            const currentKeywordsJson = editGroup.keywords || '';
            if (keywordsJson !== currentKeywordsJson) {
                updates.keywords = keywordsJson;
            }

            // 检查匹配模式变更
            const currentMatchMode = editGroup.match_mode ?? 0;
            if (matchMode !== currentMatchMode) {
                updates.match_mode = matchMode;
            }

            // 只在有变更时才更新
            if (Object.keys(updates).length > 1) {
                updateGroup.mutate(updates, {
                    onSuccess: () => {
                        setGroupName('');
                        setSelectedMembers([]);
                        setKeywords([]);
                        setMatchMode(0);
                        setIsOpen(false);
                    },
                });
            } else {
                // 没有变更，直接关闭
                setIsOpen(false);
            }
        } else {
            // 创建模式
            const items: GroupItem[] = selectedMembers.map((member, index) => ({
                channel_id: member.channel_id,
                model_name: member.name,
                priority: index + 1,
                weight: member.weight ?? 1,
            }));

            createGroup.mutate(
                {
                    name: groupName,
                    mode,
                    keywords: keywordsJson,
                    match_mode: matchMode,
                    items,
                },
                {
                    onSuccess: () => {
                        setGroupName('');
                        setSelectedMembers([]);
                        setKeywords([]);
                        setMatchMode(0);
                        setIsOpen(false);
                    },
                }
            );
        }
    };

    const isValid = groupKey.length > 0 && selectedMembers.length > 0;

    return (
        <>
            <MorphingDialogTitle>
                <header className="mb-5 flex items-center justify-between">
                    <h2 className="text-2xl font-bold text-card-foreground">
                        {isEdit ? (t('edit.title') || '编辑分组') : t('create.title')}
                    </h2>
                    <MorphingDialogClose
                        className="relative right-0 top-0"
                        variants={{
                            initial: { opacity: 0, scale: 0.8 },
                            animate: { opacity: 1, scale: 1 },
                            exit: { opacity: 0, scale: 0.8 },
                        }}
                    />
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription>
                <form onSubmit={handleSubmit}>
                    <FieldGroup className="gap-4">
                        {/* Group Name */}
                        <Field>
                            <FieldLabel htmlFor="group-name">{t('form.name')}</FieldLabel>
                            <Input
                                id="group-name"
                                value={groupName}
                                onChange={(e) => setGroupName(e.target.value)}
                                className="rounded-xl"
                            />
                        </Field>

                        {/* Mode */}
                        <div className="flex gap-1">
                            {([1, 2, 3, 4] as const).map((m) => (
                                <button
                                    key={m}
                                    type="button"
                                    onClick={() => setMode(m)}
                                    className={cn(
                                        'flex-1 py-1 text-xs rounded-lg transition-colors',
                                        mode === m ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'
                                    )}
                                >
                                    {t(`mode.${m === 1 ? 'roundRobin' : m === 2 ? 'random' : m === 3 ? 'failover' : 'weighted'}`)}
                                </button>
                            ))}
                        </div>

                        {/* 关键字配置 */}
                        <KeywordConfig
                            keywords={keywords}
                            matchMode={matchMode}
                            onKeywordsChange={setKeywords}
                            onMatchModeChange={setMatchMode}
                            className="border rounded-xl p-4 bg-muted/30"
                        />

                        {/* 模型区域（包含添加功能） */}
                        <MembersSection
                            members={selectedMembers}
                            onReorder={setSelectedMembers}
                            onRemove={handleRemoveMember}
                            onWeightChange={handleWeightChange}
                            onAdd={handleAddMember}
                            onAutoAdd={handleAutoAdd}
                            removingIds={removingIds}
                            emptyText={t('form.noItems')}
                            showWeight={mode === 4}
                            autoAddDisabled={autoAddDisabled}
                        />

                        {/* Submit Button */}
                        <Button
                            type="submit"
                            disabled={!isValid || createGroup.isPending || updateGroup.isPending}
                            className="w-full rounded-xl h-11"
                        >
                            {isEdit
                                ? (updateGroup.isPending ? (t('edit.saving') || '保存中...') : (t('edit.save') || '保存'))
                                : (createGroup.isPending ? t('create.submitting') : t('create.submit'))
                            }
                        </Button>
                    </FieldGroup>
                </form>
            </MorphingDialogDescription>
        </>
    );
}
