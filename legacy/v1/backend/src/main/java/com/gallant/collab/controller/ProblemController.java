package com.gallant.collab.controller;

import com.gallant.collab.common.PageQuery;
import com.gallant.collab.common.PageResult;
import com.gallant.collab.domain.Problem;
import com.gallant.collab.dto.ProblemDtos.*;
import com.gallant.collab.service.ProblemService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/problems")
@RequiredArgsConstructor
public class ProblemController {

    private final ProblemService service;

    @GetMapping
    public PageResult<Problem> list(PageQuery pq,
                                    @RequestParam(required = false) String status,
                                    @RequestParam(required = false) String stage,
                                    @RequestParam(required = false) String priority,
                                    @RequestParam(required = false) String tab,
                                    @RequestParam(required = false) String query) {
        return service.page(pq, status, stage, priority, tab, query);
    }

    @GetMapping("/{id}")
    public ProblemDetail detail(@PathVariable String id) {
        return service.detail(id);
    }

    @PostMapping
    public Problem create(@Valid @RequestBody CreateProblemRequest req) {
        return service.create(req);
    }

    // ============================ 阶段动作 ============================

    @PostMapping("/{id}/actions/review")
    public Problem review(@PathVariable String id, @Valid @RequestBody ReviewActionRequest req) {
        return service.review(id, req);
    }

    @PostMapping("/{id}/actions/propose")
    public Problem propose(@PathVariable String id, @Valid @RequestBody ProposeActionRequest req) {
        return service.propose(id, req);
    }

    @PostMapping("/{id}/actions/meeting")
    public Problem meeting(@PathVariable String id, @Valid @RequestBody MeetingActionRequest req) {
        return service.meeting(id, req);
    }

    @PostMapping("/{id}/actions/arbitrate")
    public Problem arbitrate(@PathVariable String id, @Valid @RequestBody ArbitrateActionRequest req) {
        return service.arbitrate(id, req);
    }

    @PostMapping("/{id}/actions/consult")
    public Problem consult(@PathVariable String id, @Valid @RequestBody ConsultActionRequest req) {
        return service.consult(id, req);
    }

    @PostMapping("/{id}/actions/implement")
    public Problem implement(@PathVariable String id, @Valid @RequestBody ImplementActionRequest req) {
        return service.implement(id, req);
    }

    @PostMapping("/{id}/actions/evaluate")
    public Problem evaluate(@PathVariable String id, @Valid @RequestBody EvaluateActionRequest req) {
        return service.evaluate(id, req);
    }
}
