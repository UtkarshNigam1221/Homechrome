import { useState, useCallback } from 'react';

export function useCursorPagination(initialLimit = 20) {
  const [limit, setLimit] = useState(initialLimit);
  const [currentCursor, setCurrentCursor] = useState<string | undefined>(undefined);
  const [cursorStack, setCursorStack] = useState<string[]>([]);

  const goToNextPage = useCallback((nextCursor: string) => {
    setCursorStack(prev => [...prev, currentCursor ?? '']);
    setCurrentCursor(nextCursor);
  }, [currentCursor]);

  const goToPreviousPage = useCallback(() => {
    setCursorStack(prev => {
      const newStack = [...prev];
      const previousCursor = newStack.pop();
      setCurrentCursor(previousCursor === '' ? undefined : previousCursor);
      return newStack;
    });
  }, []);

  const resetPagination = useCallback(() => {
    setCurrentCursor(undefined);
    setCursorStack([]);
  }, []);

  const changeLimit = useCallback((newLimit: number) => {
    setLimit(newLimit);
    setCurrentCursor(undefined);
    setCursorStack([]);
  }, []);

  return {
    limit,
    cursor: currentCursor,
    hasPrevious: cursorStack.length > 0,
    goToNextPage,
    goToPreviousPage,
    resetPagination,
    changeLimit,
  };
}
