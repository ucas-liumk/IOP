package com.gallant.collab.common;

/**
 * 当前请求用户上下文。ThreadLocal 持有, 由 UserContextInterceptor 在请求开始时设置, 结束时清理。
 * 不区分角色, 仅做审计追踪用。
 */
public final class UserContext {
    private static final ThreadLocal<CurrentUser> HOLDER = new ThreadLocal<>();

    private UserContext() {}

    public static void set(CurrentUser user) {
        HOLDER.set(user);
    }

    public static CurrentUser get() {
        CurrentUser u = HOLDER.get();
        if (u == null) {
            throw new BizException(40100, "当前请求未设置用户上下文");
        }
        return u;
    }

    public static void clear() {
        HOLDER.remove();
    }

    public record CurrentUser(Long id, String name, String dept) {}
}
