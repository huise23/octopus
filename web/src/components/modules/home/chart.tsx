'use client';

import { useStatsDaily, useStatsHourly } from '@/api/endpoints/stats';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import { useMemo, useState } from 'react';
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import { useTranslations } from 'next-intl';
import { formatCount, formatMoney } from '@/lib/utils';
import dayjs from 'dayjs';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
const PERIODS = ['1', '7', '30'] as const;

export function StatsChart() {
    const { data: statsDaily } = useStatsDaily();
    const { data: statsHourly } = useStatsHourly();
    const [period, setPeriod] = useState<typeof PERIODS[number]>('1');
    const t = useTranslations('home.chart');

    const sortedDaily = useMemo(() => {
        if (!statsDaily) return [];
        return [...statsDaily].sort((a, b) => a.date.localeCompare(b.date));
    }, [statsDaily]);

    const chartData = useMemo(() => {
        if (period === '1') {
            if (!statsHourly) return [];
            return statsHourly.map((stat) => {
                return {
                    date: `${stat.hour}:00`,
                    request_count: stat.request_count.raw,
                    input_token: stat.input_token.raw,
                    output_token: stat.output_token.raw,
                    total_token: stat.total_token.raw,
                    probe_count: stat.probe_count.raw,
                    probe_total_token: stat.probe_total_token.raw,
                    total_cost: stat.total_cost.raw,
                };
            });
        } else {
            const days = parseInt(period);
            return sortedDaily.slice(-days).map((stat) => {
                return {
                    date: dayjs(stat.date).format('MM/DD'),
                    request_count: stat.request_count.raw,
                    input_token: stat.input_token.raw,
                    output_token: stat.output_token.raw,
                    total_token: stat.total_token.raw,
                    probe_count: stat.probe_count.raw,
                    probe_total_token: stat.probe_total_token.raw,
                    total_cost: stat.total_cost.raw,
                };
            });
        }
    }, [sortedDaily, statsHourly, period]);

    const totals = useMemo(() => {
        if (period === '1') {
            if (!statsHourly) return { requests: 0, cost: 0, totalToken: 0, inputToken: 0, outputToken: 0, probeCount: 0, probeToken: 0 };
            return {
                requests: statsHourly.reduce((acc, stat) => acc + stat.request_count.raw, 0),
                cost: statsHourly.reduce((acc, stat) => acc + stat.total_cost.raw, 0),
                totalToken: statsHourly.reduce((acc, stat) => acc + stat.total_token.raw, 0),
                inputToken: statsHourly.reduce((acc, stat) => acc + stat.input_token.raw, 0),
                outputToken: statsHourly.reduce((acc, stat) => acc + stat.output_token.raw, 0),
                probeCount: statsHourly.reduce((acc, stat) => acc + stat.probe_count.raw, 0),
                probeToken: statsHourly.reduce((acc, stat) => acc + stat.probe_total_token.raw, 0),
            };
        } else {
            const days = parseInt(period);
            const recentStats = sortedDaily.slice(-days);
            return {
                requests: recentStats.reduce((acc, stat) => acc + stat.request_success.raw + stat.request_failed.raw, 0),
                cost: recentStats.reduce((acc, stat) => acc + stat.total_cost.raw, 0),
                totalToken: recentStats.reduce((acc, stat) => acc + stat.total_token.raw, 0),
                inputToken: recentStats.reduce((acc, stat) => acc + stat.input_token.raw, 0),
                outputToken: recentStats.reduce((acc, stat) => acc + stat.output_token.raw, 0),
                probeCount: recentStats.reduce((acc, stat) => acc + stat.probe_count.raw, 0),
                probeToken: recentStats.reduce((acc, stat) => acc + stat.probe_total_token.raw, 0),
            };
        }
    }, [sortedDaily, statsHourly, period]);

    const chartConfig = {
        request_count: { label: t('totalRequests') },
        total_cost: { label: t('totalCost') },
        total_token: { label: t('totalToken') },
        input_token: { label: t('inputToken') },
        output_token: { label: t('outputToken') },
        probe_count: { label: t('retryCount') },
        probe_total_token: { label: t('retryToken') },
    };

    const getPeriodLabel = (p: typeof period) => {
        const labels = {
            '1': t('period.today'),
            '7': t('period.last7Days'),
            '30': t('period.last30Days'),
        };
        return labels[p];
    };

    const handlePeriodClick = () => {
        const currentIndex = PERIODS.indexOf(period);
        const nextIndex = (currentIndex + 1) % PERIODS.length;
        setPeriod(PERIODS[nextIndex]);
    };

    return (
        <div className="rounded-3xl bg-card border-card-border border pt-2 pb-0 text-card-foreground custom-shadow">
            <div className="flex justify-between items-start px-4 pb-2">
                <div className="flex flex-wrap gap-x-4 gap-y-2 text-sm">
                    <div>
                        <div className="text-xs text-muted-foreground">{t('totalRequests')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.requests).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.requests).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('totalCost')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatMoney(totals.cost).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatMoney(totals.cost).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('totalToken')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.totalToken).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.totalToken).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('inputToken')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.inputToken).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.inputToken).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('outputToken')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.outputToken).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.outputToken).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('retryCount')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.probeCount).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.probeCount).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('retryToken')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.probeToken).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.probeToken).formatted.unit}</span>
                        </div>
                    </div>
                </div>
                <div
                    className="flex gap-2 text-sm cursor-pointer hover:opacity-80 transition-opacity"
                    onClick={handlePeriodClick}
                >
                    <div>
                        <div className="text-xs text-muted-foreground">{t('timePeriod')}</div>
                        <div className="text-xl font-semibold">{getPeriodLabel(period)}</div>
                    </div>
                </div>
            </div>
            <ChartContainer config={chartConfig} className="h-40 w-full" >
                <AreaChart accessibilityLayer data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} />
                    <YAxis
                        tickLine={false}
                        axisLine={false}
                        tickFormatter={(value) => {
                            const formatted = formatMoney(value);
                            return `${formatted.formatted.value}${formatted.formatted.unit}`;
                        }}
                    />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                    <Area type="monotone" dataKey="request_count" stroke="var(--chart-1)" fillOpacity={0} />
                    <Area type="monotone" dataKey="total_cost" stroke="var(--chart-2)" fillOpacity={0} />
                    <Area type="monotone" dataKey="total_token" stroke="var(--chart-3)" fillOpacity={0} />
                    <Area type="monotone" dataKey="input_token" stroke="var(--chart-4)" fillOpacity={0} />
                    <Area type="monotone" dataKey="output_token" stroke="var(--chart-5)" fillOpacity={0} />
                    <Area type="monotone" dataKey="probe_count" stroke="var(--chart-1)" strokeDasharray="4 4" fillOpacity={0} />
                    <Area type="monotone" dataKey="probe_total_token" stroke="var(--chart-2)" strokeDasharray="4 4" fillOpacity={0} />
                </AreaChart>
            </ChartContainer>
        </div>
    );
}
