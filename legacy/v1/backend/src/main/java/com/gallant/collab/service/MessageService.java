package com.gallant.collab.service;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.gallant.collab.common.UserContext;
import com.gallant.collab.domain.Message;
import com.gallant.collab.mapper.MessageMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.time.LocalDateTime;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

@Service
@RequiredArgsConstructor
public class MessageService {

    private static final Pattern MENTION = Pattern.compile("@([\\u4e00-\\u9fa5A-Za-z0-9_]+)");
    private final MessageMapper mapper;

    public Message post(String problemId, String content) {
        UserContext.CurrentUser u = UserContext.get();
        Message m = new Message();
        m.setProblemId(problemId);
        m.setActorUserId(u.id());
        m.setActorName(u.name());
        m.setContent(content);
        m.setMentions(extractMentions(content));
        m.setOccurredAt(LocalDateTime.now());
        m.setReadBy(List.of(u.id()));
        mapper.insert(m);
        return m;
    }

    public List<Message> list(String problemId) {
        return mapper.selectList(new QueryWrapper<Message>()
                .eq("problem_id", problemId).orderByAsc("occurred_at"));
    }

    private List<String> extractMentions(String content) {
        Matcher m = MENTION.matcher(content);
        return m.results().map(r -> "@" + r.group(1)).collect(Collectors.toList());
    }
}
