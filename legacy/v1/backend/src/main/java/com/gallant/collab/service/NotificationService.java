package com.gallant.collab.service;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.gallant.collab.common.UserContext;
import com.gallant.collab.domain.Message;
import com.gallant.collab.domain.Problem;
import com.gallant.collab.mapper.MessageMapper;
import com.gallant.collab.mapper.ProblemMapper;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

/**
 * 通知聚合: 未读 @我 留言 + 超期问题预警。前端轮询。
 */
@Service
@RequiredArgsConstructor
public class NotificationService {

    private final MessageMapper messageMapper;
    private final ProblemMapper problemMapper;

    public NotificationDigest fetch() {
        UserContext.CurrentUser u = UserContext.get();
        NotificationDigest d = new NotificationDigest();

        // @我 的未读留言
        List<Message> all = messageMapper.selectList(new QueryWrapper<Message>()
                .like("content", "@" + u.name()).orderByDesc("occurred_at"));
        for (Message m : all) {
            if (m.getReadBy() != null && m.getReadBy().contains(u.id())) continue;
            d.getMentions().add(new Item("mention", m.getProblemId(),
                    "@" + u.name() + " " + m.getContent(), m.getOccurredAt().toString()));
        }

        // 当前用户作为提报人/参与方的超期问题
        List<Problem> overdueMine = problemMapper.selectList(new QueryWrapper<Problem>()
                .eq("overdue", true).eq("submitter_id", u.id()));
        for (Problem p : overdueMine) {
            d.getOverdues().add(new Item("overdue", p.getId(),
                    "您提报的「" + p.getTitle() + "」已超期 " + p.getOverdueDays() + " 天", String.valueOf(p.getDueDate())));
        }

        // 即将到期 (3 天内)
        LocalDate soon = LocalDate.now().plusDays(3);
        List<Problem> dueSoon = problemMapper.selectList(new QueryWrapper<Problem>()
                .eq("submitter_id", u.id()).eq("overdue", false)
                .le("due_date", soon).ne("status", "done"));
        for (Problem p : dueSoon) {
            d.getDueSoon().add(new Item("due_soon", p.getId(),
                    "「" + p.getTitle() + "」即将到期 (" + p.getDueDate() + ")", String.valueOf(p.getDueDate())));
        }
        return d;
    }

    public void markMessageRead(Long messageId) {
        Message m = messageMapper.selectById(messageId);
        if (m == null) return;
        UserContext.CurrentUser u = UserContext.get();
        List<Long> read = m.getReadBy() != null ? new ArrayList<>(m.getReadBy()) : new ArrayList<>();
        if (!read.contains(u.id())) read.add(u.id());
        m.setReadBy(read);
        messageMapper.updateById(m);
    }

    @Data
    public static class NotificationDigest {
        private List<Item> mentions = new ArrayList<>();
        private List<Item> overdues = new ArrayList<>();
        private List<Item> dueSoon = new ArrayList<>();
        public int total() { return mentions.size() + overdues.size() + dueSoon.size(); }
    }

    public record Item(String type, String problemId, String text, String time) {}
}
