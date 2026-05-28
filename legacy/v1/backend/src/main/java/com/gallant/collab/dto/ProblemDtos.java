package com.gallant.collab.dto;

import com.gallant.collab.domain.*;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

/** Problem 模块的请求/响应 DTO */
public class ProblemDtos {

    @Data
    public static class CreateProblemRequest {
        @NotBlank private String title;
        @NotBlank private String description;
        @NotBlank private String category;
        @NotBlank private String priority;
        @NotNull  private LocalDate dueDate;
        private List<String> tags;
        private List<String> participants;
        private List<String> fileNames;
    }

    @Data
    public static class ReviewActionRequest {
        @NotBlank private String decision;           // approve/modify/reject
        @NotBlank private String reviewNote;
        private String handlerName;
        private String handlerDept;
        private List<String> coHandlers;
        private String priority;
        private LocalDate dueDate;
        private String assignNote;
    }

    @Data
    public static class ProposeActionRequest {
        @NotNull private Boolean hasDispute;
        private List<MeasureInput> measures;
        private List<DisputeInput> disputes;
        @NotBlank private String note;

        @Data
        public static class MeasureInput {
            private String code;
            private String title;
            private String owner;
            private Boolean hasDispute;
        }

        @Data
        public static class DisputeInput {
            private String point;
            private List<DisputePositionInput> positions;
        }

        @Data
        public static class DisputePositionInput {
            private String party;
            private String view;
        }
    }

    @Data
    public static class MeetingActionRequest {
        @NotBlank private String summary;
        private String attendees;
        private String consensus;
        private String pending;
        @NotBlank private String note;
        private Boolean advance;       // 是否本次完成会商, 推进到 arbitrate
    }

    @Data
    public static class ArbitrateActionRequest {
        @NotBlank private String arbitrator;
        @NotNull  private LocalDate date;
        @NotBlank private String overall;
        private List<DisputeResolution> resolutions;
        @NotBlank private String note;

        @Data public static class DisputeResolution {
            private Long disputeId;
            private String resolution;
        }
    }

    @Data
    public static class ConsultActionRequest {
        private List<String> audience;
        private String method;
        private LocalDate startDate;
        private LocalDate endDate;
        private String brief;
        private String revision;
        @NotBlank private String note;
        private Boolean advance;
    }

    @Data
    public static class ImplementActionRequest {
        private List<MeasureProgressInput> measureProgress;
        @NotBlank private String note;
        private Boolean advance;

        @Data public static class MeasureProgressInput {
            private Long measureId;
            private Integer progress;
            private String status;
            private String comment;
        }
    }

    @Data
    public static class EvaluateActionRequest {
        @NotBlank private String party;
        @NotNull  private BigDecimal quality;
        @NotNull  private BigDecimal speed;
        @NotNull  private BigDecimal collab;
        @NotNull  private BigDecimal satisfaction;
        private String comment;
        private Boolean archiveBestPractice;
    }

    @Data
    @NoArgsConstructor
    @AllArgsConstructor
    public static class ProblemDetail {
        private Problem problem;
        private AppUser submitter;
        private List<StageHistory> history;
        private List<Measure> measures;
        private List<DisputeWithPositions> disputes;
        private List<Message> messages;
        private List<Attachment> attachments;
        private ConsultStat consult;
        private List<Evaluation> evaluations;
    }

    @Data
    @NoArgsConstructor
    @AllArgsConstructor
    public static class DisputeWithPositions {
        private Dispute dispute;
        private List<DisputePosition> positions;
    }
}
