import { timestampDate } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { countBy } from "lodash-es";
import { useMemo } from "react";
import { useMemos } from "@/hooks/useMemoQueries";
import { useAllUserStats, useUserStats } from "@/hooks/useUserQueries";
import type { UserStats } from "@/types/proto/api/v1/user_service_pb";
import type { StatisticsData } from "@/types/statistics";

export interface FilteredMemoStats {
  statistics: StatisticsData;
  tags: Record<string, number>;
  loading: boolean;
}

export interface UseFilteredMemoStatsOptions {
  userName?: string;
  includeAllUsers?: boolean;
}

const aggregateUserStats = (statsList: UserStats[]) => {
  const displayTimeList: Date[] = [];
  const tagCount: Record<string, number> = {};

  for (const userStats of statsList) {
    for (const timestamp of userStats.memoDisplayTimestamps) {
      displayTimeList.push(timestampDate(timestamp));
    }

    for (const [tag, count] of Object.entries(userStats.tagCount)) {
      tagCount[tag] = (tagCount[tag] || 0) + count;
    }
  }

  return {
    activityStats: countBy(displayTimeList.map((date) => dayjs(date).format("YYYY-MM-DD"))),
    tagCount,
  };
};

export const useFilteredMemoStats = (options: UseFilteredMemoStatsOptions = {}): FilteredMemoStats => {
  const { userName, includeAllUsers = false } = options;

  // Fetch user stats if userName is provided
  const { data: userStats, isLoading: isLoadingUserStats } = useUserStats(userName);

  // Fetch full public stats for Explore without relying on paginated memo lists
  const { data: allUserStats, isLoading: isLoadingAllUserStats } = useAllUserStats({ enabled: includeAllUsers });

  // Fetch memos for fallback computation (or when userName is not provided)
  const { data: memosResponse, isLoading: isLoadingMemos } = useMemos({}, { enabled: !includeAllUsers && !userName });

  const data = useMemo(() => {
    const loading = isLoadingUserStats || isLoadingAllUserStats || isLoadingMemos;
    let activityStats: Record<string, number> = {};
    let tagCount: Record<string, number> = {};

    // Use all-user backend stats for Explore so the calendar is not limited by list pagination.
    if (includeAllUsers && allUserStats) {
      const aggregatedStats = aggregateUserStats(allUserStats);
      activityStats = aggregatedStats.activityStats;
      tagCount = aggregatedStats.tagCount;
    } else if (userName && userStats) {
      // Try to use backend user stats if userName is provided and available.
      const aggregatedStats = aggregateUserStats([userStats]);
      activityStats = aggregatedStats.activityStats;
      tagCount = aggregatedStats.tagCount;
    } else if (memosResponse?.memos) {
      // Fallback: compute from memos if backend stats not available
      // Also used for Explore and Archived contexts
      const displayTimeList: Date[] = [];
      const memos = memosResponse.memos;

      for (const memo of memos) {
        // Collect display timestamps for activity calendar
        const displayTime = memo.displayTime ? timestampDate(memo.displayTime) : undefined;
        if (displayTime) {
          displayTimeList.push(displayTime);
        }
        // Count tags
        if (memo.tags && memo.tags.length > 0) {
          for (const tag of memo.tags) {
            tagCount[tag] = (tagCount[tag] || 0) + 1;
          }
        }
      }

      activityStats = countBy(displayTimeList.map((date) => dayjs(date).format("YYYY-MM-DD")));
    }

    return {
      statistics: { activityStats },
      tags: tagCount,
      loading,
    };
  }, [includeAllUsers, allUserStats, userName, userStats, memosResponse, isLoadingUserStats, isLoadingAllUserStats, isLoadingMemos]);

  return data;
};
